// Code generated by thriftrw-plugin-yarpc
// @generated

package extendonlyserver

import (
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/baseserviceserver"
)

// Interface is the server-side interface for the ExtendOnly service.
type Interface interface {
	baseserviceserver.Interface
}

// New prepares an implementation of the ExtendOnly service for
// registration.
//
// 	handler := ExtendOnlyHandler{}
// 	dispatcher.Register(extendonlyserver.New(handler))
func New(impl Interface, opts ...thrift.RegisterOption) []transport.Procedure {

	service := thrift.Service{
		Name:    "ExtendOnly",
		Methods: []thrift.Method{},
	}

	procedures := make([]transport.Procedure, 0, 0)

	procedures = append(
		procedures,
		baseserviceserver.New(
			impl,
			append(
				opts,
				thrift.Named("ExtendOnly"),
			)...,
		)...,
	)

	procedures = append(procedures, thrift.BuildProcedures(service, opts...)...)
	return procedures
}

type handler struct{ impl Interface }
