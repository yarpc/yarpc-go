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
	"reflect"
	"strings"

	"github.com/gogo/protobuf/proto"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/procedure"
	"go.uber.org/yarpc/yarpcerrors"
)

const (
	// Encoding is the name of this encoding.
	Encoding transport.Encoding = "proto"

	// JSONEncoding is the name of the JSON encoding.
	//
	// Protobuf handlers are able to handle both Encoding and JSONEncoding encodings.
	JSONEncoding transport.Encoding = "json"
)

// UseJSON says to use the json encoding for client/server communication.
var UseJSON ClientOption = useJSON{}

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
			transport.Procedure{
				Name:        procedure.ToName(serviceName, methodName),
				HandlerSpec: transport.NewUnaryHandlerSpec(unaryHandler),
				Encoding:    JSONEncoding,
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
			transport.Procedure{
				Name:        procedure.ToName(serviceName, methodName),
				HandlerSpec: transport.NewOnewayHandlerSpec(onewayHandler),
				Encoding:    JSONEncoding,
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

// ClientOption is an option for a new Client.
type ClientOption interface {
	apply(*client)
}

// NewClient creates a new client.
func NewClient(serviceName string, clientConfig transport.ClientConfig, options ...ClientOption) Client {
	return newClient(serviceName, clientConfig, options...)
}

// UnaryHandlerParams contains the parameters for creating a new UnaryHandler.
type UnaryHandlerParams struct {
	Handle     func(context.Context, proto.Message) (proto.Message, error)
	NewRequest func() proto.Message
}

// NewUnaryHandler returns a new UnaryHandler.
func NewUnaryHandler(params UnaryHandlerParams) transport.UnaryHandler {
	return newUnaryHandler(params.Handle, params.NewRequest)
}

// OnewayHandlerParams contains the parameters for creating a new OnewayHandler.
type OnewayHandlerParams struct {
	Handle     func(context.Context, proto.Message) error
	NewRequest func() proto.Message
}

// NewOnewayHandler returns a new OnewayHandler.
func NewOnewayHandler(params OnewayHandlerParams) transport.OnewayHandler {
	return newOnewayHandler(params.Handle, params.NewRequest)
}

// ClientBuilderOptions returns ClientOptions that yarpc.InjectClients should use for a
// specific client given information about the field into which the client is being injected.
func ClientBuilderOptions(_ transport.ClientConfig, structField reflect.StructField) []ClientOption {
	var opts []ClientOption
	for _, opt := range uniqueLowercaseStrings(strings.Split(structField.Tag.Get("proto"), ",")) {
		switch opt {
		case "json":
			opts = append(opts, UseJSON)
		}
	}
	return opts
}

// CastError returns an error saying that generated code could not properly cast a proto.Message to it's expected type.
func CastError(expectedType proto.Message, actualType proto.Message) error {
	return yarpcerrors.InternalErrorf("expected proto.Message to have type %T but had type %T", expectedType, actualType)
}

type useJSON struct{}

func (useJSON) apply(client *client) {
	client.encoding = JSONEncoding
}
func uniqueLowercaseStrings(s []string) []string {
	m := make(map[string]bool, len(s))
	for _, e := range s {
		if e != "" {
			m[strings.ToLower(e)] = true
		}
	}
	c := make([]string, 0, len(m))
	for key := range m {
		c = append(c, key)
	}
	return c
}
