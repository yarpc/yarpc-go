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

package oneway

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/http"
)

const callBackAddrHeader = "callBackAddr"

type onewayHandler struct {
	httpTransport *http.Transport
}

// EchoRaw implements the echo/raw procedure.
func (o *onewayHandler) EchoRaw(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) error {
	o.callHome(ctx, reqMeta, body, raw.Encoding)
	return nil
}

type jsonToken struct{ Token string }

// EchoJSON implements the echo/json procedure.
func (o *onewayHandler) EchoJSON(ctx context.Context, reqMeta yarpc.ReqMeta, token *jsonToken) error {
	o.callHome(ctx, reqMeta, []byte(token.Token), json.Encoding)
	return nil
}

// Echo implements the Oneway::Echo procedure.
func (o *onewayHandler) Echo(ctx context.Context, reqMeta yarpc.ReqMeta, Token *string) error {
	o.callHome(ctx, reqMeta, []byte(*Token), thrift.Encoding)
	return nil
}

// callHome extracts the call back address from headers, and makes a raw HTTP
// request using the same context and body
func (o *onewayHandler) callHome(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte, encoding transport.Encoding) {
	// reduce the chance of a race condition
	time.Sleep(time.Millisecond * 100)

	callBackAddr, exists := reqMeta.Headers().Get(callBackAddrHeader)
	if !exists {
		panic("could not find callBackAddr in headers")
	}

	out := o.httpTransport.NewSingleOutbound("http://" + callBackAddr)
	out.Start()
	defer out.Stop()

	_, err := out.CallOneway(ctx, &transport.Request{
		Caller:    "oneway-server",
		Service:   "oneway-client",
		Procedure: "call-back",
		Encoding:  raw.Encoding,
		Body:      bytes.NewReader(body),
	})

	if err != nil {
		panic(fmt.Sprintf("could not make call back to client: %s", err))
	}
}
