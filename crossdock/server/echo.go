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

package server

import (
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/encoding/thrift"
)

// EchoRaw implements the echo/raw procedure.
func EchoRaw(req *raw.Request, body []byte) ([]byte, *raw.Response, error) {
	return body, nil, nil
}

// EchoJSON implements the echo procedure.
func EchoJSON(req *json.Request, body map[string]interface{}) (map[string]interface{}, *json.Response, error) {
	return body, nil, nil
}

// EchoThrift implements the Thrift Echo service.
type EchoThrift struct{}

// Echo endpoint for the Echo service.
func (EchoThrift) Echo(req *thrift.Request, ping *echo.Ping) (*echo.Pong, *thrift.Response, error) {
	return &echo.Pong{Boop: ping.Beep}, nil, nil
}
