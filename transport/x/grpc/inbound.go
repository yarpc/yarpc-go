// Copyright (c) 2017 Uber Technologies, Inc.
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
	"errors"
	"net"
	"sync"

	"google.golang.org/grpc"

	"go.uber.org/yarpc/api/transport"
	internalsync "go.uber.org/yarpc/internal/sync"
)

var (
	errRouterNotSet = errors.New("router not set")

	_ transport.Inbound = (*Inbound)(nil)
)

// Inbound is a grpc transport.Inbound.
type Inbound struct {
	once    internalsync.LifecycleOnce
	lock    sync.Mutex
	address string
	router  transport.Router
	server  *grpc.Server
}

// NewInbound returns a new Inbound for the given address.
func NewInbound(address string) *Inbound {
	return &Inbound{internalsync.Once(), sync.Mutex{}, address, nil, nil}
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
	return i.IsRunning()
}

// SetRouter implements transport.Inbound#SetRouter.
func (i *Inbound) SetRouter(router transport.Router) {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.router = router
}

// Transports implements transport.Inbound#Transports.
func (i *Inbound) Transports() []transport.Transport {
	return []transport.Transport{}
}

func (i *Inbound) start() error {
	i.lock.Lock()
	defer i.lock.Unlock()
	if i.router == nil {
		return errRouterNotSet
	}
	server := grpc.NewServer(grpc.CustomCodec(noopCodec{}))
	if err := registerProcedures(server, i.router.Procedures()); err != nil {
		return err
	}
	listener, err := net.Listen("tcp", i.address)
	if err != nil {
		return err
	}
	go func() {
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
	return nil
}

// TODO
func registerProcedures(server *grpc.Server, procedures []transport.Procedure) error {
	return nil
}
