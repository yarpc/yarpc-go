// Copyright (c) 2018 Uber Technologies, Inc.
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
	"go.uber.org/yarpc/internal/introspection"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/zap"
)

// ChannelInbound receives YARPC requests over TChannel.
// It may be constructed using the NewInbound method on ChannelTransport.
// If you have a YARPC peer.Chooser, use the unqualified tchannel.Transport
// instead (instead of the tchannel.ChannelTransport).
type ChannelInbound struct {
	transport *ChannelTransport

	once *lifecycle.Once
}

// NewInbound returns a new TChannel inbound backed by a shared TChannel
// transport.  The returned ChannelInbound does not support peer.Chooser
// and uses TChannel's own internal load balancing peer selection.
// If you have a YARPC peer.Chooser, use the unqualified tchannel.NewInbound
// instead.
// There should only be one inbound for TChannel since all outbounds send the
// listening port over non-ephemeral connections so a service can deduplicate
// locally- and remotely-initiated persistent connections.
func (t *ChannelTransport) NewInbound() *ChannelInbound {
	return &ChannelInbound{
		once:      lifecycle.NewOnce(),
		transport: t,
	}
}

// SetRouter configures a router to handle incoming requests.
// This satisfies the transport.Inbound interface, and would be called
// by a dispatcher when it starts.
func (i *ChannelInbound) SetRouter(router transport.Router) {
	i.transport.router = router
}

// Transports returns a slice containing the ChannelInbound's underlying
// ChannelTransport.
func (i *ChannelInbound) Transports() []transport.Transport {
	return []transport.Transport{i.transport}
}

// Channel returns the underlying Channel for this Inbound.
func (i *ChannelInbound) Channel() Channel {
	return i.transport.ch
}

// Start starts this Inbound. Note that this does not start listening for
// connections; that occurs when you start the underlying ChannelTransport is
// started.
func (i *ChannelInbound) Start() error {
	return i.once.Start(func() error {
		i.transport.logger.Info("started TChannel inbound", zap.String("address", i.transport.ListenAddr()))
		if i.transport.router == nil || len(i.transport.router.Procedures()) == 0 {
			i.transport.logger.Warn("no procedures specified for tchannel inbound")
		}
		return nil
	})
}

// Stop stops the TChannel outbound. This currently does nothing.
func (i *ChannelInbound) Stop() error {
	return i.once.Stop(nil)
}

// IsRunning returns whether the ChannelInbound is running.
func (i *ChannelInbound) IsRunning() bool {
	return i.once.IsRunning()
}

// Introspect returns the state of the inbound for introspection purposes.
func (i *ChannelInbound) Introspect() introspection.InboundStatus {
	c := i.transport.Channel()
	stateString := ""
	if c != nil {
		stateString = c.State().String()
	}
	return introspection.InboundStatus{
		Transport: "tchannel",
		Endpoint:  i.transport.ListenAddr(),
		State:     stateString,
	}
}
