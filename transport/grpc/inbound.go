package grpc

import (
	"fmt"
	"net"
	"net/url"

	"go.uber.org/yarpc/transport"

	"google.golang.org/grpc"
)

// Inbound is a gRPC Inbound.
type Inbound interface {
	transport.Inbound

	Server() *grpc.Server
}

// NewInbound builds a new gRPC Inbound.
func NewInbound(port int) Inbound {
	i := &inbound{port: port}
	return i
}

type inbound struct {
	port   int
	server *grpc.Server
}

func (i *inbound) Server() *grpc.Server {
	return i.server
}

// gRPC expects a service and a server to have the same interface when it's configured.
// For our purposes, we are faking the interfaces and forwarding all requests directly to
// the YARPC gRPC Handle instead
type passThroughService interface{}
type passThroughServer struct{}

func (i *inbound) Start(service transport.ServiceDetail, d transport.Deps) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", i.port))
	if err != nil {
		return err
	}

	// Use a codec that passes through the bytes from gRPC requests to YARPC encoders
	// TODO customize the Codec.String() function which is used for the Content-Type header
	i.server = grpc.NewServer(grpc.CustomCodec(passThroughCodec{}))

	gHandler := handler{
		Registry: service.Registry,
		Deps:     d,
	}

	var serviceDescs []grpc.ServiceDesc

	// TODO generate serviceDescs from yarpc registration info
	serviceDescs = append(serviceDescs, grpc.ServiceDesc{
		ServiceName: url.QueryEscape("yarpc"),
		HandlerType: (*passThroughService)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: url.QueryEscape("yarpc"), // TODO: Is this what we want here?
				Handler:    gHandler.Handle,          // grpc.methodHandler
			},
		},
		Streams: []grpc.StreamDesc{},
	})

	// TODO Generate the serviceDesc from the configuration of the dispatcher name and procedure names

	// Register Services
	for _, desc := range serviceDescs {
		i.server.RegisterService(&desc, passThroughServer{})
	}

	// TODO should block until ready to accept requests
	go i.server.Serve(lis)

	return nil
}

func (i *inbound) Stop() error {
	if i.server == nil {
		return nil
	}
	i.server.GracefulStop()
	return nil
}
