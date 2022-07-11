// Copyright (c) 2022 Uber Technologies, Inc.
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

package grpc

import (
	"net"
	"sync"

	"go.uber.org/yarpc/api/transport"
	yarpctls "go.uber.org/yarpc/api/transport/tls"
	"go.uber.org/yarpc/api/x/introspection"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/transport/internal/tlsmux"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	errRouterNotSet = yarpcerrors.Newf(yarpcerrors.CodeInternal, "router not set")

	_ introspection.IntrospectableInbound = (*Inbound)(nil)
	_ transport.Inbound                   = (*Inbound)(nil)
)

// Inbound is a grpc transport.Inbound.
type Inbound struct {
	once     *lifecycle.Once
	lock     sync.RWMutex
	t        *Transport
	listener net.Listener
	options  *inboundOptions
	router   transport.Router
	server   *grpc.Server
}

// newInbound returns a new Inbound for the given listener.
func newInbound(t *Transport, listener net.Listener, options ...InboundOption) *Inbound {
	return &Inbound{
		once:     lifecycle.NewOnce(),
		t:        t,
		listener: listener,
		options:  newInboundOptions(options),
	}
}

// Start implements transport.Lifecycle#Start.
func (i *Inbound) Start() error {
	return i.once.Start(i.start)
}

// Stop implements transport.Lifecycle#Stop.
func (i *Inbound) Stop() error {
	return i.once.Stop(i.stop)
}

// IsRunning implements transport.Lifecycle#IsRunning.
func (i *Inbound) IsRunning() bool {
	return i.once.IsRunning()
}

// SetRouter implements transport.Inbound#SetRouter.
func (i *Inbound) SetRouter(router transport.Router) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.router = router
}

// Addr returns the address on which the server is listening.
//
// Returns nil if Start has not been called yet
func (i *Inbound) Addr() net.Addr {
	i.lock.RLock()
	defer i.lock.RUnlock()
	// i.server is set in start, so checking against nil checks
	// if Start has been called
	// we check if i.listener is nil just for safety
	if i.server == nil || i.listener == nil {
		return nil
	}
	return i.listener.Addr()
}

// Transports implements transport.Inbound#Transports.
func (i *Inbound) Transports() []transport.Transport {
	return []transport.Transport{i.t}
}

func (i *Inbound) start() error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.router == nil {
		return errRouterNotSet
	}

	handler := newHandler(i, i.t.options.logger)

	serverOptions := []grpc.ServerOption{
		grpc.CustomCodec(customCodec{}),
		grpc.UnknownServiceHandler(handler.handle),
		grpc.MaxRecvMsgSize(i.t.options.serverMaxRecvMsgSize),
		grpc.MaxSendMsgSize(i.t.options.serverMaxSendMsgSize),
	}

	listener := i.listener

	if i.options.creds != nil {
		serverOptions = append(serverOptions, grpc.Creds(i.options.creds))
	} else if i.options.tlsMode != yarpctls.Disabled {
		listener = tlsmux.NewListener(tlsmux.Config{
			Listener:      listener,
			TLSConfig:     i.options.tlsConfig.Clone(),
			Logger:        i.t.options.logger,
			Meter:         i.t.options.meter,
			ServiceName:   i.t.options.serviceName,
			TransportName: TransportName,
			Mode:          i.options.tlsMode,
		})
	}

	if i.t.options.serverMaxHeaderListSize != nil {
		serverOptions = append(serverOptions, grpc.MaxHeaderListSize(*i.t.options.serverMaxHeaderListSize))
	}

	server := grpc.NewServer(serverOptions...)

	go func() {
		i.t.options.logger.Info("started GRPC inbound", zap.Stringer("address", i.listener.Addr()))
		if len(i.router.Procedures()) == 0 {
			i.t.options.logger.Warn("no procedures specified for GRPC inbound")
		}
		// TODO there should be some mechanism to block here
		// there is a race because the listener gets set in the grpc
		// Server implementation and we should be able to block
		// until Serve initialization is done
		//
		// It would be even better if we could do this outside the
		// lock in i
		//
		// TODO Server always returns a non-nil error but should
		// we do something with some or all errors?
		_ = server.Serve(listener)
	}()
	i.server = server
	return nil
}

func (i *Inbound) stop() error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.server != nil {
		i.server.GracefulStop()
	}
	i.server = nil
	return nil
}

// Introspect returns the current state of the inbound.
func (i *Inbound) Introspect() introspection.InboundStatus {
	state := "Stopped"
	if i.IsRunning() {
		state = "Started"
	}
	var addrString string
	if addr := i.Addr(); addr != nil {
		addrString = addr.String()
	}
	return introspection.InboundStatus{
		Transport: TransportName,
		Endpoint:  addrString,
		State:     state,
	}
}
