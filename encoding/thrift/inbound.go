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

package thrift

import (
	"bytes"
	"io/ioutil"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"

	"github.com/thriftrw/thriftrw-go/protocol"
	"github.com/thriftrw/thriftrw-go/wire"
	"golang.org/x/net/context"
)

// thriftHandler wraps a Thrift Handler into a transport.Handler
type thriftHandler struct {
	Handler  Handler
	Protocol protocol.Protocol
}

func (t thriftHandler) Handle(ctx context.Context, treq *transport.Request) (*transport.Response, error) {
	body, err := ioutil.ReadAll(treq.Body)
	if err != nil {
		return nil, err
	}

	value, err := t.Protocol.Decode(bytes.NewReader(body), wire.TStruct)
	if err != nil {
		return nil, decodeError{Reason: err}
	}

	service, method := splitProcedure(treq.Procedure)
	res, err := t.Handler.Handle(ctx, &Request{
		Service: service,
		Method:  method,
		Meta:    yarpc.NewMeta(treq.Headers),
		Body:    value,
	})

	var headers map[string]string
	if res.Meta != nil {
		headers = res.Meta.Headers()
	}

	var buffer bytes.Buffer
	if err := t.Protocol.Encode(res.Body, &buffer); err != nil {
		return nil, encodeError{Reason: err}
	}

	// TODO change transport to use ResponseWriter-like API to avoid coying like
	// this.
	rbody := ioutil.NopCloser(&buffer)
	return &transport.Response{Headers: headers, Body: rbody}, nil
}
