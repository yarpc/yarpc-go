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

package json

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

// Client TODO
type Client interface {
	// Call performs an outbound JSON request.
	//
	// responseOut is a pointer to a value that can be filled with
	// json.Unmarshal.
	//
	// Returns the response metadata or an error if the request failed.
	Call(ctx context.Context, req *Request, responseOut interface{}) (yarpc.Meta, error)
}

// Request represents an outbound JSON request.
type Request struct {
	// Name of the procedure being called.
	Procedure string

	// Request metadata
	Meta yarpc.Meta

	// Request body. This may be any type that can be serialized by
	// json.Marshal.
	Body interface{}

	// TTL is the ttl in ms
	TTL time.Duration
}

// New builds a new JSON client.
func New(c transport.Channel) Client {
	return jsonClient{
		t:       c.Outbound,
		caller:  c.Caller,
		service: c.Service,
	}
}

type jsonClient struct {
	t transport.Outbound

	caller, service string
}

func (c jsonClient) Call(ctx context.Context, req *Request, responseOut interface{}) (yarpc.Meta, error) {
	encoded, err := json.Marshal(req.Body)
	if err != nil {
		return nil, marshalError{Reason: err}
	}

	var headers transport.Headers
	if req.Meta != nil {
		headers = req.Meta.Headers()
	}

	treq := transport.Request{
		Caller:    c.caller,
		Service:   c.service,
		Procedure: req.Procedure,
		Headers:   headers,
		Body:      bytes.NewReader(encoded),
		TTL:       req.TTL, // TODO consider default
	}

	tres, err := c.t.Call(ctx, &treq)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(tres.Body)
	if err := dec.Decode(responseOut); err != nil {
		return nil, unmarshalError{Reason: err}
	}

	if err := tres.Body.Close(); err != nil {
		return nil, err
	}

	return yarpc.NewMeta(tres.Headers), nil
}
