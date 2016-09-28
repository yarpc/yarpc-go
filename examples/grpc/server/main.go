package main

import (
	"fmt"
	"log"
	"net"

	gr "github.com/yarpc/yarpc-go/transport/grpc"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	port = ":50054"
)

// @generated
type service interface {
	Bar(ctx context.Context, in *[]byte) (*[]byte, error)
}

// @generated
// dec is testCodec.Unmarshal
func handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	fmt.Println("grpc.methodHandler::main.handler")
	var in []byte
	if err := dec(&in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(service).Bar(ctx, &in) // &in
	}
	// TODO this path hasn't been exercised
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/foo/bar",
	}
	// this is a grpc.UnaryHandler
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		fmt.Println("grpc.UnaryHandler::main.handler")

		// call the users handler, casting the req to the right type
		return srv.(service).Bar(ctx, req.(*[]byte))
	}
	return interceptor(ctx, in, info, handler)
}

// @generated
var serviceDesc = grpc.ServiceDesc{
	ServiceName: "foo",
	HandlerType: (*service)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "bar",
			Handler:    handler, // grpc.methodHandler
		},
	},
	Streams: []grpc.StreamDesc{},
}

type server struct{}

func (server) Bar(ctx context.Context, in *[]byte) (*[]byte, error) {
	fmt.Println("main.server::Bar")
	res := []byte("server says hi")
	if in != nil {
		res = []byte(fmt.Sprintf("server got request body: %s", string(*in)))
	}
	return &res, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// TODO only 1 codec is supported at the moment, https://github.com/grpc/grpc-go/issues/803
	s := grpc.NewServer(grpc.CustomCodec(gr.PassThroughCodec{}))
	s.RegisterService(&serviceDesc, server{}) // @generated
	s.Serve(lis)
}
