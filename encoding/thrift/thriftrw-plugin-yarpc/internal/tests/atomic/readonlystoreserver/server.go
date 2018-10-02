// Code generated by thriftrw-plugin-yarpc
// @generated

package readonlystoreserver

import (
	"context"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/baseserviceserver"
)

// Interface is the server-side interface for the ReadOnlyStore service.
type Interface interface {
	baseserviceserver.Interface

	Integer(
		ctx context.Context,
		Key *string,
	) (int64, error)
}

// New prepares an implementation of the ReadOnlyStore service for
// registration.
//
// 	handler := ReadOnlyStoreHandler{}
// 	dispatcher.Register(readonlystoreserver.New(handler))
func New(impl Interface, opts ...thrift.RegisterOption) []transport.Procedure {
	h := handler{impl}
	service := thrift.Service{
		Name: "ReadOnlyStore",
		Methods: []thrift.Method{

			thrift.Method{
				Name: "integer",
				HandlerSpec: thrift.HandlerSpec{

					Type:  transport.Unary,
					Unary: thrift.UnaryHandler(h.Integer),
				},
				Signature:    "Integer(Key *string) (int64)",
				ThriftModule: atomic.ThriftModule,
			},
		},
	}

	procedures := make([]transport.Procedure, 0, 1)

	procedures = append(
		procedures,
		baseserviceserver.New(
			impl,
			append(
				opts,
				thrift.Named("ReadOnlyStore"),
			)...,
		)...,
	)
	procedures = append(procedures, thrift.BuildProcedures(service, opts...)...)
	return procedures
}

type handler struct{ impl Interface }

func (h handler) Integer(ctx context.Context, body wire.Value) (thrift.Response, error) {
	var args atomic.ReadOnlyStore_Integer_Args
	if err := args.FromWire(body); err != nil {
		return thrift.Response{}, err
	}

	success, err := h.impl.Integer(ctx, args.Key)

	hadError := err != nil
	result, err := atomic.ReadOnlyStore_Integer_Helper.WrapResponse(success, err)

	var response thrift.Response
	if err == nil {
		response.IsApplicationError = hadError
		response.Body = result
	}
	return response, err
}
