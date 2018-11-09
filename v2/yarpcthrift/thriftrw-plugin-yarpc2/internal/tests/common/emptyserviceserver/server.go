// Code generated by thriftrw-plugin-yarpc
// @generated

package emptyserviceserver

import (
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcthrift"
)

// Interface is the server-side interface for the EmptyService service.
type Interface interface {
}

// New prepares an implementation of the EmptyService service for
// registration.
//
// 	handler := EmptyServiceHandler{}
// 	dispatcher.Register(emptyserviceserver.New(handler))
func New(impl Interface, opts ...yarpcthrift.RegisterOption) []yarpc.EncodingProcedure {

	service := yarpcthrift.Service{
		Name:    "EmptyService",
		Methods: []yarpcthrift.Method{},
	}

	procedures := make([]yarpc.EncodingProcedure, 0, 0)
	procedures = append(procedures, yarpcthrift.BuildProcedures(service, opts...)...)
	return procedures
}

type handler struct{ impl Interface }
