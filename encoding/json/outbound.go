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

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/internal/encoding"
	"github.com/yarpc/yarpc-go/internal/meta"
	"github.com/yarpc/yarpc-go/transport"
)

// Client makes JSON requests to a single service.
type Client interface {
	// Call performs an outbound JSON request.
	//
	// resBodyOut is a pointer to a value that can be filled with
	// json.Unmarshal.
	//
	// Returns the response or an error if the request failed.
	Call(reqMeta yarpc.CallReqMeta, reqBody interface{}, resBodyOut interface{}) (yarpc.CallResMeta, error)
}

// New builds a new JSON client.
func New(c transport.Channel) Client {
	return jsonClient{ch: c}
}

type jsonClient struct {
	ch transport.Channel
}

func (c jsonClient) Call(reqMeta yarpc.CallReqMeta, reqBody interface{}, resBodyOut interface{}) (yarpc.CallResMeta, error) {
	treq := transport.Request{
		Caller:   c.ch.Caller(),
		Service:  c.ch.Service(),
		Encoding: Encoding,
	}
	ctx := meta.ToTransportRequest(reqMeta, &treq)

	encoded, err := json.Marshal(reqBody)
	if err != nil {
		return nil, encoding.RequestBodyEncodeError(&treq, err)
	}

	treq.Body = bytes.NewReader(encoded)
	tres, err := c.ch.GetOutbound().Call(ctx, &treq)
	// TODO: transport must return response context

	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(tres.Body)
	if err := dec.Decode(resBodyOut); err != nil {
		return nil, encoding.ResponseBodyDecodeError(&treq, err)
	}

	if err := tres.Body.Close(); err != nil {
		return nil, err
	}

	// TODO: when transport returns response context, use that here.
	return meta.FromTransportResponse(ctx, tres), nil
}
