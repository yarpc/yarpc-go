package grpc

import (
	"fmt"
	"net"

	"github.com/yarpc/yarpc-go/transport"

	"net/url"

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

// gRPC expects a service and a server to have the same interface when it's configured.
// For our purposes, we are faking the interfaces and forwarding all requests directly to
// the YARPC gRPC Handle instead
type passThroughService interface{}
type passThroughServer struct{}

func (i *inbound) Start(h transport.Handler, d transport.Deps) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", i.port))
	if err != nil {
		return err
	}

	// Use a codec that passes through the bytes from gRPC requests to YARPC encoders
	// TODO customize the Codec.String() function which is used for the Content-Type header
	i.server = grpc.NewServer(grpc.CustomCodec(PassThroughCodec{}))

	gHandler := handler{
		Handler: h,
		Deps:    d,
	}

	var serviceDescs []grpc.ServiceDesc
	if reg, ok := h.(transport.Registry); ok {
		serviceProcedures := make(map[string][]string)
		for _, sp := range reg.ServiceProcedures() {
			serviceProcedures[sp.Service] = append(serviceProcedures[sp.Service], sp.Procedure)
		}

		for service, procs := range serviceProcedures {
			serviceDescs = append(serviceDescs, *createServiceDesc(gHandler, service, procs))
		}
	} else {
		// Called with a plain Handler instead of Dispatcher. Fall back to no meaningful service description.
		serviceDescs = append(serviceDescs, *createServiceDesc(gHandler, "yarpc", []string{"yarpc"}))
	}

	// Register Services
	for _, desc := range serviceDescs {
		i.server.RegisterService(&desc, passThroughServer{})
	}

	// TODO should block until ready to accept requests
	go i.server.Serve(lis)

	return nil
}

func createServiceDesc(gHandler handler, service string, procedures []string) *grpc.ServiceDesc {
	methodDescs := make([]grpc.MethodDesc, 0, len(procedures))
	for _, proc := range procedures {
		methodDescs = append(methodDescs, grpc.MethodDesc{
			MethodName: url.QueryEscape(proc),
			Handler:    gHandler.Handle,
		})
	}

	return &grpc.ServiceDesc{
		ServiceName: url.QueryEscape(service),
		HandlerType: (*passThroughService)(nil),
		Methods:     methodDescs,
		Streams:     []grpc.StreamDesc{},
	}
}

func (i *inbound) Stop() error {
	if i.server == nil {
		return nil
	}
	i.server.GracefulStop()
	return nil
}
