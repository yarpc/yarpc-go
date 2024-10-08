// Copyright (c) 2024 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package tchannel

import (
	"errors"
	"strings"
	"sync"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/zap"
)

const (
	tchannelTracingKeyPrefix      = "$tracing$"
	tchannelTracingKeyMappingSize = 100
)

var errChannelOrServiceNameIsRequired = errors.New(
	"cannot instantiate tchannel.ChannelTransport: " +
		"please provide a service name with the ServiceName option " +
		"or an existing Channel with the WithChannel option")

// NewChannelTransport is a YARPC transport that facilitates sending and
// receiving YARPC requests through TChannel. It uses a shared TChannel
// Channel for both, incoming and outgoing requests, ensuring reuse of
// connections and other resources.
//
// Either the local service name (with the ServiceName option) or a user-owned
// TChannel (with the WithChannel option) MUST be specified.
//
// ChannelTransport uses the underlying TChannel Channel for load balancing
// and peer managament.
// Use NewTransport and its NewOutbound to support YARPC peer.Choosers.
func NewChannelTransport(opts ...TransportOption) (*ChannelTransport, error) {
	var options transportOptions
	options.tracer = opentracing.GlobalTracer()
	for _, opt := range opts {
		opt(&options)
	}

	// Attempt to construct a channel on behalf of the caller if none given.
	// Defer the error until Start since NewChannelTransport does not have
	// an error return.
	var err error
	ch := options.ch

	if ch == nil {
		if options.name == "" {
			err = errChannelOrServiceNameIsRequired
		} else {
			opts := tchannel.ChannelOptions{Tracer: options.tracer}
			ch, err = tchannel.NewChannel(options.name, &opts)
			options.ch = ch
		}
	}

	if err != nil {
		return nil, err
	}

	return options.newChannelTransport(), nil
}

func (options transportOptions) newChannelTransport() *ChannelTransport {
	logger := options.logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ChannelTransport{
		once:              lifecycle.NewOnce(),
		ch:                options.ch,
		addr:              options.addr,
		tracer:            options.tracer,
		logger:            logger.Named("tchannel"),
		originalHeaders:   options.originalHeaders,
		newResponseWriter: newHandlerWriter,
	}
}

// ChannelTransport maintains TChannel peers and creates inbounds and outbounds for
// TChannel.
// If you have a YARPC peer.Chooser, use the unqualified tchannel.Transport
// instead.
type ChannelTransport struct {
	once              *lifecycle.Once
	ch                Channel
	addr              string
	tracer            opentracing.Tracer
	logger            *zap.Logger
	router            transport.Router
	originalHeaders   bool
	newResponseWriter func(inboundCallResponse, tchannel.Format, headerCase) responseWriter
}

// Channel returns the underlying TChannel "Channel" instance.
func (t *ChannelTransport) Channel() Channel {
	return t.ch
}

// ListenAddr exposes the listen address of the transport.
func (t *ChannelTransport) ListenAddr() string {
	return t.addr
}

// Start starts the TChannel transport. This starts making connections and
// accepting inbound requests. All inbounds must have been assigned a router
// to accept inbound requests before this is called.
func (t *ChannelTransport) Start() error {
	return t.once.Start(t.start)
}

func (t *ChannelTransport) start() error {

	if t.router != nil {
		// Set up handlers. This must occur after construction because the
		// dispatcher, or its equivalent, calls SetRouter before Start.
		// This also means that SetRouter should be called on every inbound
		// before calling Start on any transport or inbound.
		services := make(map[string]struct{})
		for _, p := range t.router.Procedures() {
			services[p.Service] = struct{}{}
		}

		for s := range services {
			sc := t.ch.GetSubChannel(s)
			existing := sc.GetHandlers()
			sc.SetHandler(handler{existing: existing, router: t.router, tracer: t.tracer, logger: t.logger, newResponseWriter: t.newResponseWriter})
		}
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
func (t *ChannelTransport) Stop() error {
	return t.once.Stop(t.stop)
}

func (t *ChannelTransport) stop() error {
	t.ch.Close()
	return nil
}

// IsRunning returns whether the ChannelTransport is running.
func (t *ChannelTransport) IsRunning() bool {
	return t.once.IsRunning()
}

// GetPropagationFormat returns the opentracing propagation depends on transport.
// For TChannel, the format is opentracing.TextMap
// For HTTP and gRPC, the format is opentracing.HTTPHeaders
func GetPropagationFormat(transport string) opentracing.BuiltinFormat {
	if transport == "tchannel" {
		return opentracing.TextMap
	}
	return opentracing.HTTPHeaders
}

// PropagationCarrier is an interface to combine both reader and writer interface
type PropagationCarrier interface {
	opentracing.TextMapReader
	opentracing.TextMapWriter
}

// GetPropagationCarrier get the propagation carrier depends on the transport.
// The carrier is used for accessing the transport headers.
// For TChannel, a special carrier is used. For details, see comments of TChannelHeadersCarrier
func GetPropagationCarrier(headers map[string]string, transport string) PropagationCarrier {
	if transport == "tchannel" {
		return TChannelHeadersCarrier(headers)
	}
	return opentracing.TextMapCarrier(headers)
}

// TChannelHeadersCarrier is a dedicated carrier for TChannel.
// When writing the tracing headers into headers, the $tracing$ prefix is added to each tracing header key.
// When reading the tracing headers from headers, the $tracing$ prefix is removed from each tracing header key.
type TChannelHeadersCarrier map[string]string

var _ PropagationCarrier = TChannelHeadersCarrier{}

// ForeachKey iterates over all tracing headers in the carrier, applying the provided
// handler function to each header after stripping the $tracing$ prefix from the keys.
func (c TChannelHeadersCarrier) ForeachKey(handler func(string, string) error) error {
	for k, v := range c {
		if !strings.HasPrefix(k, tchannelTracingKeyPrefix) {
			continue
		}
		noPrefixKey := tchannelTracingKeyDecoding.mapAndCache(k)
		if err := handler(noPrefixKey, v); err != nil {
			return err
		}
	}
	return nil
}

// Set adds a tracing header to the carrier, prefixing the key with $tracing$ before storing it.
func (c TChannelHeadersCarrier) Set(key, value string) {
	prefixedKey := tchannelTracingKeyEncoding.mapAndCache(key)
	c[prefixedKey] = value
}

// tchannelTracingKeysMapping is to optimize the efficiency of tracing header key manipulations.
// The implementation is forked from tchannel-go: https://github.com/uber/tchannel-go/blob/dev/tracing_keys.go#L36
type tchannelTracingKeysMapping struct {
	sync.RWMutex
	mapping map[string]string
	mapper  func(key string) string
}

var tchannelTracingKeyEncoding = &tchannelTracingKeysMapping{
	mapping: make(map[string]string),
	mapper: func(key string) string {
		return tchannelTracingKeyPrefix + key
	},
}

var tchannelTracingKeyDecoding = &tchannelTracingKeysMapping{
	mapping: make(map[string]string),
	mapper: func(key string) string {
		return key[len(tchannelTracingKeyPrefix):]
	},
}

func (m *tchannelTracingKeysMapping) mapAndCache(key string) string {
	m.RLock()
	v, ok := m.mapping[key]
	m.RUnlock()
	if ok {
		return v
	}
	m.Lock()
	defer m.Unlock()
	if v, ok := m.mapping[key]; ok {
		return v
	}
	mappedKey := m.mapper(key)
	if len(m.mapping) < tchannelTracingKeyMappingSize {
		m.mapping[key] = mappedKey
	}
	return mappedKey
}
