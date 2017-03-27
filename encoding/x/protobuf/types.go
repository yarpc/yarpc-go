// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package protobuf

import (
	"context"
	"fmt"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/procedure"

	"github.com/gogo/protobuf/proto"
)

// Encoding is the name of this encoding.
const Encoding transport.Encoding = "protobuf"

// GetApplicationError returns the application error from the server, if present.
//
// TODO: this has overlap with IsApplicationError
func GetApplicationError(err error) error {
	if applicationError, ok := err.(*applicationError); ok {
		return applicationError
	}
	return nil
}

// ***all below functions should only be called by generated code***

// BuildProcedures builds the transport.Procedures.
func BuildProcedures(
	serviceName string,
	methodNameToUnaryHandler map[string]transport.UnaryHandler,
	methodNameToOnewayHandler map[string]transport.OnewayHandler,
) []transport.Procedure {
	procedures := make([]transport.Procedure, 0, len(methodNameToUnaryHandler))
	for methodName, unaryHandler := range methodNameToUnaryHandler {
		procedures = append(
			procedures,
			transport.Procedure{
				Name:        procedure.ToName(serviceName, methodName),
				HandlerSpec: transport.NewUnaryHandlerSpec(unaryHandler),
				Encoding:    Encoding,
			},
		)
	}
	for methodName, onewayHandler := range methodNameToOnewayHandler {
		procedures = append(
			procedures,
			transport.Procedure{
				Name:        procedure.ToName(serviceName, methodName),
				HandlerSpec: transport.NewOnewayHandlerSpec(onewayHandler),
				Encoding:    Encoding,
			},
		)
	}
	return procedures
}

// Client is a protobuf client.
type Client interface {
	Call(
		ctx context.Context,
		requestMethodName string,
		request proto.Message,
		newResponse func() proto.Message,
		options ...yarpc.CallOption,
	) (proto.Message, error)
	CallOneway(
		ctx context.Context,
		requestMethodName string,
		request proto.Message,
		options ...yarpc.CallOption,
	) (transport.Ack, error)
}

// NewClient creates a new client.
func NewClient(serviceName string, clientConfig transport.ClientConfig) Client {
	return newClient(serviceName, clientConfig)
}

// NewUnaryHandler returns a new UnaryHandler.
func NewUnaryHandler(
	handle func(context.Context, proto.Message) (proto.Message, error),
	newRequest func() proto.Message,
) transport.UnaryHandler {
	return newUnaryHandler(handle, newRequest)
}

// NewOnewayHandler returns a new OnewayHandler.
func NewOnewayHandler(
	handleOneway func(context.Context, proto.Message) error,
	newRequest func() proto.Message,
) transport.OnewayHandler {
	return newOnewayHandler(handleOneway, newRequest)
}

// CastError returns an error saying that generated code could not properly cast a proto.Message to it's expected type.
func CastError(expectedType proto.Message, actualType proto.Message) error {
	return fmt.Errorf("expected proto.Message to have type %T but had type %T", expectedType, actualType)
}
