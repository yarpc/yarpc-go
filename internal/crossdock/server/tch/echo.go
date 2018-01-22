// Copyright (c) 2018 Uber Technologies, Inc.
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

package tch

import (
	"github.com/uber/tchannel-go/json"
	"github.com/uber/tchannel-go/raw"
	"github.com/uber/tchannel-go/thrift"
	"go.uber.org/yarpc/internal/crossdock/thrift/gen-go/echo"
	"golang.org/x/net/context"
)

type echoRawHandler struct{}

func (echoRawHandler) Handle(ctx context.Context, args *raw.Args) (*raw.Res, error) {
	return &raw.Res{Arg2: args.Arg2, Arg3: args.Arg3}, nil
}

func (echoRawHandler) OnError(ctx context.Context, err error) {
	onError(ctx, err)
}

func echoJSONHandler(ctx json.Context, body map[string]interface{}) (map[string]interface{}, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return body, nil
}

type echoThriftHandler struct{}

func (h *echoThriftHandler) Echo(ctx thrift.Context, ping *echo.Ping) (*echo.Pong, error) {
	ctx.SetResponseHeaders(ctx.Headers())
	return &echo.Pong{Boop: ping.Beep}, nil
}
