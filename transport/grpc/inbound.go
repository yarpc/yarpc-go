package grpc

import (
	"fmt"
	"net"

	"golang.org/x/net/context"

	"github.com/yarpc/yarpc-go/transport"

	"google.golang.org/grpc"
)

// Inbound is a GRPC Inbound.
type Inbound interface {
	transport.Inbound
}

// NewInbound builds a new GRPC Inbound.
func NewInbound(port int) Inbound {
	i := &inbound{port: port}
	return i
}

type inbound struct {
	port int
}

func (i *inbound) Start(h transport.Handler, d transport.Deps) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", i.port))
	if err != nil {
		return err
	}

	// TODO need to get all supported encodings...
	// TODO need to a get a list of all procedure names...

	// TODO only 1 codec is supported at the moment, https://github.com/grpc/grpc-go/issues/803
	s := grpc.NewServer(grpc.CustomCodec(RawCodec{}))

	// TODO should block until ready to accept requests
	go s.Serve(lis)

	return nil
}

func (inbound) Stop() error {
	return nil
}

func handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	return nil, nil
}
