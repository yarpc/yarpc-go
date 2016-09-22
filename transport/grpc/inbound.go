package grpc

import (
	"fmt"
	"net"

	"github.com/yarpc/yarpc-go/transport"

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
	if i.server == nil {
		return nil
	}
	return i.server
}

// gRPC expects a service and a server to have the same interface when it's configured
// For our purposes, we are faking the interfaces and forwarding all requests directly to
// the YARPC gRPC Handle instead
type passThroughService interface{}
type passThroughServer struct{}

func (i *inbound) Start(h transport.Handler, d transport.Deps) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", i.port))
	if err != nil {
		return err
	}

	// TODO need to get all supported encodings...
	// TODO need to a get a list of all procedure names...

	// Use a codec that passes through the bytes from gRPC requests to YARPC encoders
	i.server = grpc.NewServer(grpc.CustomCodec(PassThroughCodec{}))

	gHandler := handler{
		Handler: h,
		Deps:    d,
	}

	// TODO Generate the serviceDesc from the configuration of the dispatcher name and procedure names
	var serviceDesc = grpc.ServiceDesc{
		ServiceName: "foo",
		HandlerType: (*passThroughService)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "bar",
				Handler:    gHandler.Handle, // grpc.methodHandler
			},
		},
		Streams: []grpc.StreamDesc{},
	}

	// Register Service
	i.server.RegisterService(&serviceDesc, passThroughServer{})

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
