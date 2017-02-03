package tchannel

import (
	"fmt"
	"sync"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	intsync "go.uber.org/yarpc/internal/sync"
	"go.uber.org/yarpc/peer/hostport"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
)

// Transport is a TChannel transport suitable for use with YARPC's peer
// selection system.
// The transport implements peer.Transport so multiple peer.List
// implementations can retain and release shared peers.
// The transport implements transport.Transport so it is suitable for lifecycle
// management.
type Transport struct {
	lock sync.Mutex
	once intsync.LifecycleOnce

	ch     Channel
	router transport.Router
	tracer opentracing.Tracer
	addr   string

	peers map[string]*hostport.Peer
}

// NewTransport is a YARPC transport that facilitates sending and receiving
// YARPC requests through TChannel.
// It uses a shared TChannel Channel for both, incoming and outgoing requests,
// ensuring reuse of connections and other resources.
//
// Either the local service name (with the ServiceName option) or a user-owned
// TChannel (with the WithChannel option) MUST be specified.
func NewTransport(opts ...TransportOption) (*Transport, error) {
	var config transportConfig
	config.tracer = opentracing.GlobalTracer()
	for _, opt := range opts {
		opt(&config)
	}

	// Attempt to construct a channel on behalf of the caller if none given.
	// Defer the error until Start since NewChannelTransport does not have
	// an error return.
	var err error

	if config.ch != nil {
		return nil, fmt.Errorf("NewTransport does not accept WithChannel, use NewChannelTransport")
	}
	// if config.name == "" {
	// 	return nil, errChannelOrServiceNameIsRequired
	// }

	chopts := tchannel.ChannelOptions{Tracer: config.tracer}
	ch, err := tchannel.NewChannel(config.name, &chopts)
	if err != nil {
		return nil, err
	}

	return &Transport{
		ch:     ch,
		addr:   config.addr,
		tracer: config.tracer,
		peers:  make(map[string]*hostport.Peer),
	}, err
}

// ListenAddr exposes the listen address of the transport.
func (t *Transport) ListenAddr() string {
	return t.addr
}

// RetainPeer adds a peer subscriber (typically a peer chooser) and causes the
// transport to maintain persistent connections with that peer.
func (t *Transport) RetainPeer(pid peer.Identifier, sub peer.Subscriber) (peer.Peer, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	hppid, ok := pid.(hostport.PeerIdentifier)
	if !ok {
		return nil, peer.ErrInvalidPeerType{
			ExpectedType:   "hostport.PeerIdentifier",
			PeerIdentifier: pid,
		}
	}

	p := t.getOrCreatePeer(hppid)
	p.Subscribe(sub)
	return p, nil
}

// **NOTE** should only be called while the lock write mutex is acquired
func (t *Transport) getOrCreatePeer(pid hostport.PeerIdentifier) *hostport.Peer {
	if p, ok := t.peers[pid.Identifier()]; ok {
		return p
	}

	p := hostport.NewPeer(pid, t)
	p.SetStatus(peer.Available)

	t.peers[p.Identifier()] = p

	return p
}

// ReleasePeer releases a peer from the peer.Subscriber and removes that peer
// from the Transport if nothing is listening to it.
func (t *Transport) ReleasePeer(pid peer.Identifier, sub peer.Subscriber) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	p, ok := t.peers[pid.Identifier()]
	if !ok {
		return peer.ErrTransportHasNoReferenceToPeer{
			TransportName:  "tchannel.Transport",
			PeerIdentifier: pid.Identifier(),
		}
	}

	if err := p.Unsubscribe(sub); err != nil {
		return err
	}

	if p.NumSubscribers() == 0 {
		delete(t.peers, pid.Identifier())
	}

	return nil
}

// Start starts the TChannel transport. This starts making connections and
// accepting inbound requests. All inbounds must have been assigned a router
// to accept inbound requests before this is called.
func (t *Transport) Start() error {
	return t.once.Start(t.start)
}

func (t *Transport) start() error {

	if t.router != nil {
		// Set up handlers. This must occur after construction because the
		// dispatcher, or its equivalent, calls SetRouter before Start.
		// This also means that SetRouter should be called on every inbound
		// before calling Start on any transport or inbound.
		sc := t.ch.GetSubChannel(t.ch.ServiceName())
		existing := sc.GetHandlers()
		sc.SetHandler(handler{existing: existing, router: t.router, tracer: t.tracer})
	}

	if t.ch.State() == tchannel.ChannelListening {
		// Channel.Start() was called before RPC.Start(). We still want to
		// update the Handler and what t.addr means, but nothing else.
		t.addr = t.ch.PeerInfo().HostPort
		return nil
	}

	// Default to ListenIP if addr wasn't given.
	addr := t.addr
	if addr == "" {
		listenIP, err := tchannel.ListenIP()
		if err != nil {
			return err
		}

		addr = listenIP.String() + ":0"
		// TODO(abg): Find a way to export this to users
	}

	// TODO(abg): If addr was just the port (":4040"), we want to use
	// ListenIP() + ":4040" rather than just ":4040".

	if err := t.ch.ListenAndServe(addr); err != nil {
		return err
	}

	t.addr = t.ch.PeerInfo().HostPort
	return nil
}

// Stop stops the TChannel transport. It starts rejecting incoming requests
// and draining connections before closing them.
// In a future version of YARPC, Stop will block until the underlying channel
// has closed completely.
func (t *Transport) Stop() error {
	return t.once.Stop(t.stop)
}

func (t *Transport) stop() error {
	t.ch.Close()
	return nil
}

// IsRunning returns whether the TChannel transport is running.
func (t *Transport) IsRunning() bool {
	return t.once.IsRunning()
}
