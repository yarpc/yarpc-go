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

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"

	"google.golang.org/grpc"

	"go.uber.org/yarpc/api/transport"
	internalsync "go.uber.org/yarpc/internal/sync"
)

var (
	errRouterNotSet          = errors.New("router not set")
	errRouterHasNoProcedures = errors.New("router has no procedures")

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
	serviceDescs, err := getServiceDescs(i.router)
	if err != nil {
		return err
	}
	server := grpc.NewServer(
		grpc.CustomCodec(customCodec{}),
		// TODO: does this actually work for yarpc
		// this needs a lot of review
		// TODO: always global tracer?
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(opentracing.GlobalTracer())),
	)
	for _, serviceDesc := range serviceDescs {
		server.RegisterService(serviceDesc, noopGrpcStruct{})
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

func getServiceDescs(router transport.Router) ([]*grpc.ServiceDesc, error) {
	// TODO: router.Procedures() is not guaranteed to be immutable
	procedures := router.Procedures()
	if len(procedures) == 0 {
		return nil, errRouterHasNoProcedures
	}
	serviceNameToServiceDesc := make(map[string]*grpc.ServiceDesc)
	for _, procedure := range procedures {
		serviceName, methodDesc, err := getServiceNameAndMethodDesc(router, procedure)
		if err != nil {
			return nil, err
		}
		serviceDesc, ok := serviceNameToServiceDesc[serviceName]
		if !ok {
			serviceDesc = &grpc.ServiceDesc{
				ServiceName: serviceName,
				HandlerType: (*noopGrpcInterface)(nil),
			}
			serviceNameToServiceDesc[serviceName] = serviceDesc
		}
		serviceDesc.Methods = append(serviceDesc.Methods, methodDesc)
	}
	serviceDescs := make([]*grpc.ServiceDesc, 0, len(serviceNameToServiceDesc))
	for _, serviceDesc := range serviceNameToServiceDesc {
		serviceDescs = append(serviceDescs, serviceDesc)
	}
	return serviceDescs, nil
}

func getServiceNameAndMethodDesc(router transport.Router, procedure transport.Procedure) (string, grpc.MethodDesc, error) {
	serviceName, methodName, err := procedureNameToServiceNameMethodName(procedure.Name)
	if err != nil {
		return "", grpc.MethodDesc{}, err
	}
	return serviceName, grpc.MethodDesc{
		MethodName: methodName,
		// TODO: what if two procedures have the same serviceName and methodName, but a different service?
		Handler: newMethodHandler(procedure.Service, serviceName, methodName, router).handle,
	}, nil
}

type noopGrpcInterface interface{}
type noopGrpcStruct struct{}
