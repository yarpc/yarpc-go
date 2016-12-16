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
	"sync"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"

	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/atomic"
)

// ChannelInbound is a TChannel Inbound backed by a pre-existing TChannel
// Channel.
type ChannelInbound struct {
	ch        Channel
	addr      string
	router    transport.Router
	tracer    opentracing.Tracer
	transport *ChannelTransport

	started   atomic.Bool
	startOnce sync.Once
	startErr  error
	stopped   atomic.Bool
}

// NewInbound returns a new TChannel inbound backed by a shared TChannel
// transport.  The returned ChannelInbound does not support peer.Chooser
// and uses TChannel's own internal load balancing peer selection.
func (t *ChannelTransport) NewInbound() *ChannelInbound {
	return &ChannelInbound{
		ch:        t.ch,
		tracer:    t.tracer,
		transport: t,
	}
}

// SetRouter configures a router to handle incoming requests.
// This satisfies the transport.Inbound interface, and would be called
// by a dispatcher when it starts.
func (i *ChannelInbound) SetRouter(router transport.Router) {
	i.router = router
}

// Transports returns a slice containing the ChannelInbound's underlying
// ChannelTransport.
func (i *ChannelInbound) Transports() []transport.Transport {
	return []transport.Transport{i.transport}
}

// Channel returns the underlying Channel for this Inbound.
func (i *ChannelInbound) Channel() Channel {
	return i.ch
}

// Start starts a TChannel inbound. This arranges for registration of the
// request handler based on the router given by SetRouter.
//
// Start does not start listening sockets. That occurs when you Start the
// underlying ChannelTransport.
func (i *ChannelInbound) Start() error {
	i.startOnce.Do(func() {
		i.started.Store(true)
		i.startErr = i.start()
	})

	return i.startErr
}

func (i *ChannelInbound) start() error {
	if i.router == nil {
		return errors.ErrNoRouter
	}

	// Set up handlers. This must occur after construction because the
	// dispatcher, or its equivalent, calls SetRouter before Start.
	// This also means that starting inbounds should block starting the transport.
	sc := i.ch.GetSubChannel(i.ch.ServiceName())
	existing := sc.GetHandlers()
	sc.SetHandler(handler{existing: existing, router: i.router, tracer: i.tracer})

	return nil
}

// Stop stops the TChannel outbound. This currently does nothing.
func (i *ChannelInbound) Stop() error {
	i.stopped.Store(true)
	return nil
}

// IsRunning returns whether the ChannelInbound is running.
func (i *ChannelInbound) IsRunning() bool {
	return i.started.Load() && !i.stopped.Load()
}
