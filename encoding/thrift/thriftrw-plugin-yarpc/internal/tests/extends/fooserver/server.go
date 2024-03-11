// Code generated by thriftrw-plugin-yarpc
// @generated

package fooserver

import (
	transport "go.uber.org/yarpc/api/transport"
	thrift "go.uber.org/yarpc/encoding/thrift"
	nameserver "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/extends/nameserver"
)

// Interface is the server-side interface for the Foo service.
type Interface interface {
	nameserver.Interface
}

// New prepares an implementation of the Foo service for
// registration.
//
//	handler := FooHandler{}
//	dispatcher.Register(fooserver.New(handler))
func New(impl Interface, opts ...thrift.RegisterOption) []transport.Procedure {

	service := thrift.Service{
		Name:    "Foo",
		Methods: []thrift.Method{},
	}

	procedures := make([]transport.Procedure, 0, 0)

	procedures = append(
		procedures,
		nameserver.New(
			impl,
			append(
				opts,
				thrift.Named("Foo"),
			)...,
		)...,
	)
	procedures = append(procedures, thrift.BuildProcedures(service, opts...)...)
	return procedures
}
