// Copyright (c) 2016 Uber Technologies, Inc.
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

package marshal

import (
	"bytes"
	"io/ioutil"

	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/marshal/internal"
	"go.uber.org/yarpc/transport"

	"github.com/opentracing/opentracing-go"
)

// MarshalRPC encodes a SpanContext and Request into bytes
func MarshalRPC(tracer opentracing.Tracer, spanContext opentracing.SpanContext, req *transport.Request) ([]byte, error) {
	spanBytes, err := transport.MarshalSpanContext(tracer, spanContext)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	rpc := internal.RPC{
		SpanContext: spanBytes,

		CallerName:      req.Caller,
		ServiceName:     req.Service,
		Encoding:        string(req.Encoding),
		Procedure:       req.Procedure,
		Headers:         req.Headers.Items(),
		ShardKey:        &req.ShardKey,
		RoutingKey:      &req.RoutingKey,
		RoutingDelegate: &req.RoutingDelegate,
		Body:            body,
	}

	wireValue, err := rpc.ToWire()
	if err != nil {
		return nil, err
	}

	var writer bytes.Buffer
	err = protocol.Binary.Encode(wireValue, &writer)
	return writer.Bytes(), err
}

// UnmarshalRPC decodes bytes into a SpanContext and Request
func UnmarshalRPC(tracer opentracing.Tracer, request []byte) (opentracing.SpanContext, *transport.Request, error) {
	reader := bytes.NewReader(request)
	wireValue, err := protocol.Binary.Decode(reader, wire.TStruct)
	if err != nil {
		return nil, nil, err
	}

	var rpc internal.RPC
	if err = rpc.FromWire(wireValue); err != nil {
		return nil, nil, err
	}

	req := transport.Request{
		Caller:    rpc.CallerName,
		Service:   rpc.ServiceName,
		Encoding:  transport.Encoding(rpc.Encoding),
		Procedure: rpc.Procedure,
		Headers:   transport.HeadersFromMap(rpc.Headers),
		Body:      bytes.NewBuffer(rpc.Body),
	}

	if rpc.ShardKey != nil {
		req.ShardKey = *rpc.ShardKey
	}
	if rpc.RoutingKey != nil {
		req.RoutingKey = *rpc.RoutingKey
	}
	if rpc.RoutingDelegate != nil {
		req.RoutingDelegate = *rpc.RoutingDelegate
	}

	spanContext, err := transport.UnmarshalSpanContext(tracer, rpc.SpanContext)
	if err != nil {
		return nil, nil, err
	}

	return spanContext, &req, nil
}
