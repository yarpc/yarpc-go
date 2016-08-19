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

package raw

import (
	"bytes"
	"io/ioutil"

	"golang.org/x/net/context"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/internal/meta"
	"github.com/yarpc/yarpc-go/transport"
)

// Client makes Raw requests to a single service.
type Client interface {
	// Call performs an outbound Raw request.
	Call(ctx context.Context, reqMeta yarpc.CallReqMeta, body []byte) ([]byte, yarpc.CallResMeta, error)
}

// New builds a new Raw client.
func New(c transport.Channel) Client {
	return rawClient{ch: c}
}

type rawClient struct {
	ch transport.Channel
}

func (c rawClient) Call(ctx context.Context, reqMeta yarpc.CallReqMeta, body []byte) ([]byte, yarpc.CallResMeta, error) {
	treq := transport.Request{
		Caller:   c.ch.Caller(),
		Service:  c.ch.Service(),
		Encoding: Encoding,
		Body:     bytes.NewReader(body),
	}
	meta.ToTransportRequest(reqMeta, &treq)

	tres, err := c.ch.GetOutbound().Call(ctx, &treq)
	if err != nil {
		return nil, nil, err
	}
	defer tres.Body.Close()

	resBody, err := ioutil.ReadAll(tres.Body)
	if err != nil {
		return nil, nil, err
	}

	return resBody, meta.FromTransportResponse(tres), nil
}
