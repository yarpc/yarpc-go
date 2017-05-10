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

	"go.uber.org/yarpc/api/transport"
	internalsync "go.uber.org/yarpc/internal/sync"

	"google.golang.org/grpc"
)

var (
	errRouterNotSet          = errors.New("router not set")
	errRouterHasNoProcedures = errors.New("router has no procedures")

	_ transport.Inbound = (*Inbound)(nil)
)

// Inbound is a grpc transport.Inbound.
type Inbound struct {
	once           internalsync.LifecycleOnce
	lock           sync.Mutex
	listener       net.Listener
	inboundOptions *inboundOptions
	router         transport.Router
	server         *grpc.Server
}

// NewInbound returns a new Inbound for the given listener.
func NewInbound(listener net.Listener, options ...InboundOption) *Inbound {
	return &Inbound{internalsync.Once(), sync.Mutex{}, listener, newInboundOptions(options), nil, nil}
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
	serviceDescs, err := i.getServiceDescs()
	if err != nil {
		return err
	}
	server := grpc.NewServer(
		grpc.CustomCodec(customCodec{}),
		// TODO: does this actually work for yarpc
		// this needs a lot of review
		//grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(i.inboundOptions.getTracer())),

		// TODO grpc.UnaryInterceptor handles when parameter is nil, but should not rely on this
		grpc.UnaryInterceptor(i.inboundOptions.getUnaryInterceptor()),
	)
	for _, serviceDesc := range serviceDescs {
		server.RegisterService(serviceDesc, noopGrpcStruct{})
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
		_ = server.Serve(i.listener)
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

func (i *Inbound) getServiceDescs() ([]*grpc.ServiceDesc, error) {
	// TODO: router.Procedures() is not guaranteed to be immutable
	// https://github.com/yarpc/yarpc-go/issues/825
	procedures := i.router.Procedures()
	if len(procedures) == 0 {
		return nil, errRouterHasNoProcedures
	}
	grpcServiceNameToServiceDesc := make(map[string]*grpc.ServiceDesc)
	for _, procedure := range procedures {
		serviceName, methodDesc, err := i.getServiceNameAndMethodDesc(procedure)
		if err != nil {
			return nil, err
		}
		serviceDesc, ok := grpcServiceNameToServiceDesc[serviceName]
		if !ok {
			serviceDesc = &grpc.ServiceDesc{
				ServiceName: serviceName,
				HandlerType: (*noopGrpcInterface)(nil),
			}
			grpcServiceNameToServiceDesc[serviceName] = serviceDesc
		}
		serviceDesc.Methods = append(serviceDesc.Methods, methodDesc)
	}
	serviceDescs := make([]*grpc.ServiceDesc, 0, len(grpcServiceNameToServiceDesc))
	for _, serviceDesc := range grpcServiceNameToServiceDesc {
		serviceDescs = append(serviceDescs, serviceDesc)
	}
	return serviceDescs, nil
}

func (i *Inbound) getServiceNameAndMethodDesc(procedure transport.Procedure) (string, grpc.MethodDesc, error) {
	serviceName, methodName, err := procedureNameToServiceNameMethodName(procedure.Name)
	if err != nil {
		return "", grpc.MethodDesc{}, err
	}
	return serviceName, grpc.MethodDesc{
		MethodName: methodName,
		// TODO: what if two procedures have the same serviceName and methodName, but a different service?
		Handler: newHandler(
			procedure.Service,
			serviceName,
			methodName,
			procedure.Encoding,
			i.router,
		).handle,
	}, nil
}

type noopGrpcInterface interface{}
type noopGrpcStruct struct{}
