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

package transport

import (
	"context"
	"io"
	"strings"

	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap/zapcore"
)

// Request is the low level request representation.
type Request struct {
	// Name of the service making the request.
	Caller string

	// Name of the service to which the request is being made.
	// The service refers to the canonical traffic group for the service.
	Service string

	// Name of the encoding used for the request body.
	Encoding Encoding

	// Name of the procedure being called.
	Procedure string

	// Headers for the request.
	Headers Headers

	// ShardKey is an opaque string that is meaningful to the destined service
	// for how to relay a request within a cluster to the shard that owns the
	// key.
	ShardKey string

	// RoutingKey refers to a traffic group for the destined service, and when
	// present may override the service name for purposes of routing.
	RoutingKey string

	// RoutingDelegate refers to the traffic group for a service that proxies
	// for the destined service for routing purposes. The routing delegate may
	// override the routing key and service.
	RoutingDelegate string

	// Features for the request.
	Features []Feature

	// Request payload.
	Body io.Reader
}

// MarshalLogObject implements zap.ObjectMarshaler.
func (r *Request) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	// TODO (#788): Include headers once we can omit PII.
	enc.AddString("caller", r.Caller)
	enc.AddString("service", r.Service)
	enc.AddString("encoding", string(r.Encoding))
	enc.AddString("procedure", r.Procedure)
	enc.AddString("shardKey", r.ShardKey)
	enc.AddString("routingKey", r.RoutingKey)
	enc.AddString("routingDelegate", r.RoutingDelegate)
	enc.AddArray("features", zapcore.ArrayMarshalerFunc(func(arrayEncoder zapcore.ArrayEncoder) error {
		for _, feature := range r.Features {
			// We might get a feature over the wire that the client supports
			// but the server does not support, so we cannot assume that
			// MarshalText() will not return an error.
			arrayEncoder.AppendString(feature.String())
		}
		return nil
	}))
	return nil
}

// Encoding represents an encoding format for requests.
type Encoding string

// ValidateRequest validates the given request. An error is returned if the
// request is invalid.
//
// Inbound transport implementations may use this to validate requests before
// handling them. Outbound implementations don't need to validate requests;
// they are always validated before the outbound is called.
func ValidateRequest(req *Request) error {
	var missingParams []string
	if req.Service == "" {
		missingParams = append(missingParams, "service name")
	}
	if req.Procedure == "" {
		missingParams = append(missingParams, "procedure")
	}
	if req.Caller == "" {
		missingParams = append(missingParams, "caller name")
	}
	if req.Encoding == "" {
		missingParams = append(missingParams, "encoding")
	}
	if len(missingParams) > 0 {
		return yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "missing %s", strings.Join(missingParams, ", "))
	}
	return nil
}

// ValidateUnaryContext validates that a context for a unary request is valid
// and contains all required information, and returns a YARPC error with code
// yarpcerrors.CodeInvalidArgument otherwise.
func ValidateUnaryContext(ctx context.Context) error {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		return yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "missing TTL")
	}
	return nil
}
