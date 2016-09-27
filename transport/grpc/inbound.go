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
			methodDescs := make([]grpc.MethodDesc, 0, len(procs))
			for _, proc := range procs {
				methodDescs = append(methodDescs, grpc.MethodDesc{
					MethodName: proc,
					Handler:    gHandler.Handle,
				})
			}

			serviceDescs = append(serviceDescs, grpc.ServiceDesc{
				ServiceName: service,
				HandlerType: (*passThroughService)(nil),
				Methods:     methodDescs,
				Streams:     []grpc.StreamDesc{},
			})
		}
	} else {
		// Called with a plain Handler instead of Dispatcher. Fall back to no meaningful service description.
		serviceDescs = append(serviceDescs, grpc.ServiceDesc{
			// TODO: Once we figure out a way to get the service name from the Handler (we need it
			// for the TChannel inbound too), we should use that here instead.
			ServiceName: "yarpc",
			HandlerType: (*passThroughService)(nil),
			Methods: []grpc.MethodDesc{
				{
					MethodName: "yarpc",         // TODO: Is this what we want here?
					Handler:    gHandler.Handle, // grpc.methodHandler
				},
			},
			Streams: []grpc.StreamDesc{},
		})
	}

	// TODO Generate the serviceDesc from the configuration of the dispatcher name and procedure names
	// Register Service

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
