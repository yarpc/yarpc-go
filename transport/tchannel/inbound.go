// Copyright (c) 2016 Uber Technologies, Inc.
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

	"github.com/yarpc/yarpc-go/transport"

	"github.com/uber/tchannel-go"
)

// Inbound is a TChannel Inbound.
type Inbound interface {
	transport.Inbound

	// Returns the underlying Channel for this Inbound.
	Channel() *tchannel.Channel
}

// InboundOption configures Inbound.
type InboundOption func(*inbound)

// ListenAddr changes the address on which the TChannel server will listen for
// connections. By default, the server listens on an OS-assigned port.
//
// This option has no effect if the Chanel provided to NewInbound is already
// listening for connections when Start() is called.
func ListenAddr(addr string) InboundOption {
	return func(i *inbound) { i.addr = addr }
}

// NewInbound builds a new TChannel inbound from the given Channel. Existing
// methods registered on the channel remain registered and are preferred when
// a call is received.
func NewInbound(ch *tchannel.Channel, opts ...InboundOption) Inbound {
	i := &inbound{ch: ch}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

type inbound struct {
	ch       *tchannel.Channel
	addr     string
	listener net.Listener
}

func (i *inbound) Channel() *tchannel.Channel {
	return i.ch
}

func (i *inbound) Start(h transport.Handler) error {
	sc := i.ch.GetSubChannel(i.ch.ServiceName())
	existing := sc.GetHandlers()
	sc.SetHandler(handler{existing, h})

	if i.ch.State() == tchannel.ChannelListening {
		// Channel.Start() was called before RPC.Start(). We still want to
		// update the Handler and what i.addr means, but nothing else.
		i.addr = i.ch.PeerInfo().HostPort
		return nil
	}

	// Default to ListenIP if addr wasn't given.
	addr := i.addr
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

	var err error
	i.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	i.addr = i.listener.Addr().String() // in case it changed

	if err := i.ch.Serve(i.listener); err != nil {
		return err
	}

	return nil
}

func (i *inbound) Stop() error {
	i.ch.Close()
	return nil
}
