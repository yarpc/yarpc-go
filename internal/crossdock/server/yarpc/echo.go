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

package yarpc

import (
	"context"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/crossdock/crossdockpb"
	"go.uber.org/yarpc/internal/crossdock/thrift/echo"
)

// EchoRaw implements the echo/raw procedure.
func EchoRaw(ctx context.Context, body []byte) ([]byte, error) {
	call := yarpc.CallFromContext(ctx)
	for _, k := range call.HeaderNames() {
		if err := call.WriteResponseHeader(k, call.Header(k)); err != nil {
			return nil, err
		}
	}
	return body, nil
}

// EchoJSON implements the echo procedure.
func EchoJSON(ctx context.Context, body map[string]interface{}) (map[string]interface{}, error) {
	call := yarpc.CallFromContext(ctx)
	for _, k := range call.HeaderNames() {
		if err := call.WriteResponseHeader(k, call.Header(k)); err != nil {
			return nil, err
		}
	}
	return body, nil
}

// EchoThrift implements the Thrift Echo service.
type EchoThrift struct{}

// Echo endpoint for the Echo service.
func (EchoThrift) Echo(ctx context.Context, ping *echo.Ping) (*echo.Pong, error) {
	call := yarpc.CallFromContext(ctx)
	for _, k := range call.HeaderNames() {
		if err := call.WriteResponseHeader(k, call.Header(k)); err != nil {
			return nil, err
		}
	}
	return &echo.Pong{Boop: ping.Beep}, nil
}

// EchoProtobuf implements the Protobuf Echo service.
type EchoProtobuf struct{}

// Echo implements the Echo function for the Protobuf Echo service.
func (EchoProtobuf) Echo(_ context.Context, request *crossdockpb.Ping) (*crossdockpb.Pong, error) {
	if request == nil {
		return nil, nil
	}
	return &crossdockpb.Pong{Boop: request.Beep}, nil
}
