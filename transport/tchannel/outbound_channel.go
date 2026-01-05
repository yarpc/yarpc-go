// Copyright (c) 2026 Uber Technologies, Inc.
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
	"net"

	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/peer"
	"golang.org/x/net/context"
)

var _ peer.Transport = (*outboundChannel)(nil)

type dialerFunc = func(ctx context.Context, network, hostPort string) (net.Conn, error)

// outboundChannel holds a TChannel channel for outbound use only with custom
// dialer. This is used for creating TLS connections with different TLS config
// per outbound.
type outboundChannel struct {
	t      *Transport
	dialer dialerFunc
	ch     *tchannel.Channel
}

// newOutboundChannel returns outbound channel with given transport and dialer.
func newOutboundChannel(t *Transport, dialer dialerFunc) *outboundChannel {
	return &outboundChannel{
		t:      t,
		dialer: dialer,
	}
}

// RetainPeer delegates to the transport RetainPeer along with its channel.
func (o *outboundChannel) RetainPeer(pid peer.Identifier, sub peer.Subscriber) (peer.Peer, error) {
	return o.t.retainPeer(pid, sub, o.ch)
}

// ReleasePeer delegates to the transport ReleasePeer.
func (o *outboundChannel) ReleasePeer(pid peer.Identifier, sub peer.Subscriber) error {
	return o.t.ReleasePeer(pid, sub)
}

// start creates channel used for managing outbound peers.
// This is invoked by the transport when it is started.
func (o *outboundChannel) start() (err error) {
	o.ch, err = tchannel.NewChannel(o.t.name, &tchannel.ChannelOptions{
		Dialer:              o.dialer,
		OnPeerStatusChanged: o.t.onPeerStatusChanged,
		Tracer:              o.t.tracer,
	})
	return err
}

// stop closes the outbound channel stopping outbound connections.
// This is invoked by the transport when it is stopping.
func (o *outboundChannel) stop() {
	o.ch.Close()
}
