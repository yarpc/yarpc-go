// Copyright (c) 2019 Uber Technologies, Inc.
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
	"go.uber.org/yarpc/pkg/procedure"
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

// BuildProceduresParams contains the parameters for BuildProcedures.
type BuildProceduresParams struct {
	ServiceName         string
	UnaryHandlerParams  []BuildProceduresUnaryHandlerParams
	OnewayHandlerParams []BuildProceduresOnewayHandlerParams
	StreamHandlerParams []BuildProceduresStreamHandlerParams
}

// BuildProceduresUnaryHandlerParams contains the parameters for a UnaryHandler for BuildProcedures.
type BuildProceduresUnaryHandlerParams struct {
	MethodName string
	Handler    transport.UnaryHandler
}

// BuildProceduresOnewayHandlerParams contains the parameters for a OnewayHandler for BuildProcedures.
type BuildProceduresOnewayHandlerParams struct {
	MethodName string
	Handler    transport.OnewayHandler
}

// BuildProceduresStreamHandlerParams contains the parameters for a StreamHandler for BuildProcedures.
type BuildProceduresStreamHandlerParams struct {
	MethodName string
	Handler    transport.StreamHandler
}

// BuildProcedures builds the transport.Procedures.
func BuildProcedures(params BuildProceduresParams) []transport.Procedure {
	procedures := make([]transport.Procedure, 0, 2*(len(params.UnaryHandlerParams)+len(params.OnewayHandlerParams)))
	for _, unaryHandlerParams := range params.UnaryHandlerParams {
		procedures = append(
			procedures,
			transport.Procedure{
				Name:        procedure.ToName(params.ServiceName, unaryHandlerParams.MethodName),
				HandlerSpec: transport.NewUnaryHandlerSpec(unaryHandlerParams.Handler),
				Encoding:    Encoding,
			},
			transport.Procedure{
				Name:        procedure.ToName(params.ServiceName, unaryHandlerParams.MethodName),
				HandlerSpec: transport.NewUnaryHandlerSpec(unaryHandlerParams.Handler),
				Encoding:    JSONEncoding,
			},
		)
	}
	for _, onewayHandlerParams := range params.OnewayHandlerParams {
		procedures = append(
			procedures,
			transport.Procedure{
				Name:        procedure.ToName(params.ServiceName, onewayHandlerParams.MethodName),
				HandlerSpec: transport.NewOnewayHandlerSpec(onewayHandlerParams.Handler),
				Encoding:    Encoding,
			},
			transport.Procedure{
				Name:        procedure.ToName(params.ServiceName, onewayHandlerParams.MethodName),
				HandlerSpec: transport.NewOnewayHandlerSpec(onewayHandlerParams.Handler),
				Encoding:    JSONEncoding,
			},
		)
	}
	for _, streamHandlerParams := range params.StreamHandlerParams {
		procedures = append(
			procedures,
			transport.Procedure{
				Name:        procedure.ToName(params.ServiceName, streamHandlerParams.MethodName),
				HandlerSpec: transport.NewStreamHandlerSpec(streamHandlerParams.Handler),
				Encoding:    Encoding,
			},
			transport.Procedure{
				Name:        procedure.ToName(params.ServiceName, streamHandlerParams.MethodName),
				HandlerSpec: transport.NewStreamHandlerSpec(streamHandlerParams.Handler),
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

// StreamClient is a protobuf client with streaming.
type StreamClient interface {
	Client

	CallStream(
		ctx context.Context,
		requestMethodName string,
		opts ...yarpc.CallOption,
	) (*ClientStream, error)
}

// ClientOption is an option for a new Client.
type ClientOption interface {
	apply(*client)
}

// ClientParams contains the parameters for creating a new Client.
type ClientParams struct {
	ServiceName  string
	ClientConfig transport.ClientConfig
	Options      []ClientOption
}

// NewClient creates a new client.
func NewClient(params ClientParams) Client {
	return newClient(params.ServiceName, params.ClientConfig, params.Options...)
}

// NewStreamClient creates a new stream client.
func NewStreamClient(params ClientParams) StreamClient {
	return newClient(params.ServiceName, params.ClientConfig, params.Options...)
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

// StreamHandlerParams contains the parameters for creating a new StreamHandler.
type StreamHandlerParams struct {
	Handle func(*ServerStream) error
}

// NewStreamHandler returns a new StreamHandler.
func NewStreamHandler(params StreamHandlerParams) transport.StreamHandler {
	return newStreamHandler(params.Handle)
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
	return yarpcerrors.Newf(yarpcerrors.CodeInternal, "expected proto.Message to have type %T but had type %T", expectedType, actualType)
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

// ClientStream is a protobuf-specific client stream.
type ClientStream struct {
	stream *transport.ClientStream
}

// Context returns the context of the stream.
func (c *ClientStream) Context() context.Context {
	return c.stream.Context()
}

// Receive will receive a protobuf message from the client stream.
func (c *ClientStream) Receive(newMessage func() proto.Message, options ...yarpc.StreamOption) (proto.Message, error) {
	return readFromStream(context.Background(), c.stream, newMessage)
}

// Send will send a protobuf message to the client stream.
func (c *ClientStream) Send(message proto.Message, options ...yarpc.StreamOption) error {
	return writeToStream(context.Background(), c.stream, message)
}

// Close will close the protobuf stream.
func (c *ClientStream) Close(options ...yarpc.StreamOption) error {
	return c.stream.Close(context.Background())
}

// ServerStream is a protobuf-specific server stream.
type ServerStream struct {
	ctx    context.Context
	stream *transport.ServerStream
}

// Context returns the context of the stream.
func (s *ServerStream) Context() context.Context {
	return s.ctx
}

// Receive will receive a protobuf message from the server stream.
func (s *ServerStream) Receive(newMessage func() proto.Message, options ...yarpc.StreamOption) (proto.Message, error) {
	return readFromStream(context.Background(), s.stream, newMessage)
}

// Send will send a protobuf message to the server stream.
func (s *ServerStream) Send(message proto.Message, options ...yarpc.StreamOption) error {
	return writeToStream(context.Background(), s.stream, message)
}
