// Code generated by thriftrw-plugin-yarpc
// @generated

package helloserver

import (
	"context"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/examples/thrift/hello/thrift/hello"
	"go.uber.org/yarpc"
)

// Interface is the server-side interface for the Hello service.
type Interface interface {
	Echo(
		ctx context.Context,
		reqMeta yarpc.ReqMeta,
		Echo *hello.EchoRequest,
	) (*hello.EchoResponse, yarpc.ResMeta, error)
}

// New prepares an implementation of the Hello service for
// registration.
//
// 	handler := HelloHandler{}
// 	dispatcher.Register(helloserver.New(handler))
func New(impl Interface, opts ...thrift.RegisterOption) []transport.Registrant {
	h := handler{impl}
	service := thrift.Service{
		Name: "Hello",
		Methods: map[string]thrift.Handler{
			"echo": thrift.HandlerFunc(h.Echo),
		},
	}
	return thrift.BuildRegistrants(service, opts...)
}

type handler struct{ impl Interface }

func (h handler) Echo(
	ctx context.Context,
	reqMeta yarpc.ReqMeta,
	body wire.Value,
) (thrift.Response, error) {
	var args hello.Hello_Echo_Args
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, resMeta, err := h.impl.Echo(ctx, reqMeta, args.Echo)

	hadError := err != nil
	result, err := hello.Hello_Echo_Helper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Meta = resMeta
		response.Body = result
	}
	return response, err
}
