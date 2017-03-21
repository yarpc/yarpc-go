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

// Package protobuf implements Protocol Buffers encoding support for YARPC.
//
// https://developers.google.com/protocol-buffers/docs/proto3
package protobuf

import (
	"context"
	"fmt"

	"go.uber.org/yarpc/api/transport"

	"github.com/golang/protobuf/proto"
)

// Encoding is the name of this encoding.
const Encoding transport.Encoding = "protobuf"

// IsAppError returns true if an error returned from a client-side call is an application error.
func IsAppError(err error) bool {
	return false
}

// ***all below functions should only be called by generated code***

// BuildProcedures builds the transport.Procedures.
func BuildProcedures(serviceName string, methodNameToUnaryHandler map[string]UnaryHandler) []transport.Procedure {
	return buildProcedures(serviceName, methodNameToUnaryHandler)
}

// Client is a protobuf client.
type Client interface {
	Call(ctx context.Context, requestMethodName string, request proto.Message, newResponse func() proto.Message) (proto.Message, error)
}

// NewClient creates a new client.
func NewClient(serviceName string, clientConfig transport.ClientConfig) Client {
	return newClient(serviceName, clientConfig)
}

// UnaryHandler represents a protobuf unary request handler.
type UnaryHandler interface {
	// response message, application error, metadata, yarpc error
	Handle(ctx context.Context, requestMessage proto.Message) (proto.Message, error)
	NewRequest() proto.Message
}

// NewUnaryHandler returns a new UnaryHandler.
func NewUnaryHandler(
	handle func(context.Context, proto.Message) (proto.Message, error),
	newRequest func() proto.Message,
) UnaryHandler {
	return newUnaryHandler(handle, newRequest)
}

// CastError returns an error saying that generated code could not properly cast a proto.Message to it's expected type.
func CastError(expectedType proto.Message, actualType proto.Message) error {
	return fmt.Errorf("expected proto.Message to have type %T but had type %T", expectedType, actualType)
}
