package grpc

import (
	"fmt"
	"net"
	"net/url"

	"go.uber.org/yarpc/transport"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
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
	i.server = grpc.NewServer(
		grpc.CustomCodec(passThroughCodec{}),
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(d.Tracer())),
	)

	gHandler := handler{
		Registry: service.Registry,
		Deps:     d,
	}

	serviceDescs := getServiceDescs(gHandler)

	// Register Services
	for _, desc := range serviceDescs {
		i.server.RegisterService(&desc, passThroughServer{})
	}

	// TODO should block until ready to accept requests
	go i.server.Serve(lis)

	return nil
}

/*
ServiceDescs in GRPC define how clients can interact with a server and usually have well defined schemas to match
request methods with service functions.  For our purposes we are avoiding this check by dynamically creating the
serviceDescs from the YARPC registered procedures.  This allows us to see what methods are available and to create an
individual routing mechanism to each one.  All the routes will be forwarded to the same "Handler" which will do YARPC
routing to handle methods and encodings.
*/
func getServiceDescs(gHandler handler) []grpc.ServiceDesc {
	var serviceDescs []grpc.ServiceDesc

	// Create separate routes for each service & procedure
	for service, procs := range getServiceProcedures(gHandler.Registry) {
		serviceDescs = append(serviceDescs, *createServiceDesc(gHandler, service, procs))
	}

	// If no service procedures are configured insert a default serviceProcedure
	if len(serviceDescs) == 0 {
		serviceDescs = append(serviceDescs, *createServiceDesc(gHandler, "yarpc", []string{"yarpc"}))
	}

	return serviceDescs
}

func getServiceProcedures(reg transport.Registry) map[string][]string {
	// Go through the registry to find all the methods that are currently attached to inbounds
	serviceProcedures := make(map[string][]string)
	for _, sp := range reg.ServiceProcedures() {
		serviceProcedures[sp.Service] = append(serviceProcedures[sp.Service], sp.Procedure)
	}
	return serviceProcedures
}

func createServiceDesc(gHandler handler, service string, procedures []string) *grpc.ServiceDesc {
	methodDescs := make([]grpc.MethodDesc, len(procedures))
	for i, proc := range procedures {
		methodDescs[i] = grpc.MethodDesc{
			MethodName: url.QueryEscape(proc),
			Handler:    gHandler.Handle, // Catchall method that does custom YARPC routing
		}
	}

	return &grpc.ServiceDesc{
		ServiceName: url.QueryEscape(service),
		HandlerType: (*passThroughService)(nil),
		Methods:     methodDescs,
	}
}

func (i *inbound) Stop() error {
	if i.server == nil {
		return nil
	}
	i.server.GracefulStop()
	return nil
}
