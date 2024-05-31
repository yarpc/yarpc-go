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
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/x/introspection"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/zap"
)

// Inbound receives YARPC requests over TChannel. It may be constructed using
// the NewInbound method on a tchannel.Transport.
type Inbound struct {
	once      *lifecycle.Once
	transport *Transport
}

// NewInbound returns a new TChannel inbound backed by a shared TChannel
// transport.
// There should only be one inbound for TChannel since all outbounds send the
// listening port over non-ephemeral connections so a service can deduplicate
// locally- and remotely-initiated persistent connections.
func (t *Transport) NewInbound() *Inbound {
	return &Inbound{
		once:      lifecycle.NewOnce(),
		transport: t,
	}
}

// SetRouter configures a router to handle incoming requests.
// This satisfies the transport.Inbound interface, and would be called
// by a dispatcher when it starts.
func (i *Inbound) SetRouter(router transport.Router) {
	i.transport.router = router
}

// Transports returns a slice containing the Inbound's underlying
// Transport.
func (i *Inbound) Transports() []transport.Transport {
	return []transport.Transport{i.transport}
}

// Start starts this Inbound. Note that this does not start listening for
// connections; that occurs when you start the underlying ChannelTransport is
// started.
func (i *Inbound) Start() error {
	return i.once.Start(func() error {
		i.transport.logger.Info("started TChannel inbound", zap.String("address", i.transport.addr))
		if i.transport.router == nil || len(i.transport.router.Procedures()) == 0 {
			i.transport.logger.Warn("no procedures specified for tchannel inbound")
		}
		return nil
	})
}

// Stop stops the TChannel outbound. This currently does nothing.
func (i *Inbound) Stop() error {
	return i.once.Stop(nil)
}

// IsRunning returns whether the Inbound is running.
func (i *Inbound) IsRunning() bool {
	return i.once.IsRunning()
}

// Introspect returns the state of the inbound for introspection purposes.
func (i *Inbound) Introspect() introspection.InboundStatus {
	stateString := ""
	if i.transport.ch != nil {
		stateString = i.transport.ch.State().String()
	}
	return introspection.InboundStatus{
		Transport: "tchannel",
		Endpoint:  i.transport.addr,
		State:     stateString,
	}
}
