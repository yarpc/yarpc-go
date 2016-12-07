package grpc

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

/*
This needs to be in a separate file because the gRPC interface does not
follow golint rules, so we put it in this file so we can ignore it when
running go lint
*/
func (h handler) Handle(
	srv interface{},
	ctx context.Context,
	dec func(interface{}) error,
	interceptor grpc.UnaryServerInterceptor,
) (interface{}, error) {
	return h.handle(ctx, dec, interceptor)
}
