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

package yarpc

import (
	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
)

// EchoRaw implements the echo/raw procedure.
func EchoRaw(reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	resMeta := yarpc.NewResMeta(reqMeta.Context()).Headers(reqMeta.Headers())
	return body, resMeta, nil
}

// EchoJSON implements the echo procedure.
func EchoJSON(reqMeta yarpc.ReqMeta, body map[string]interface{}) (map[string]interface{}, yarpc.ResMeta, error) {
	resMeta := yarpc.NewResMeta(reqMeta.Context()).Headers(reqMeta.Headers())
	return body, resMeta, nil
}

// EchoThrift implements the Thrift Echo service.
type EchoThrift struct{}

// Echo endpoint for the Echo service.
func (EchoThrift) Echo(reqMeta yarpc.ReqMeta, ping *echo.Ping) (*echo.Pong, yarpc.ResMeta, error) {
	resMeta := yarpc.NewResMeta(reqMeta.Context()).Headers(reqMeta.Headers())
	return &echo.Pong{Boop: ping.Beep}, resMeta, nil
}
