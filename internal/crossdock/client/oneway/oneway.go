// Copyright (c) 2025 Uber Technologies, Inc.
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
	"context"

	"github.com/crossdock/crossdock-go"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/crossdock/client/dispatcher"
	"go.uber.org/yarpc/internal/crossdock/client/params"
	"go.uber.org/yarpc/internal/crossdock/client/random"
)

// Run starts the oneway behavior, testing a combination of encodings and transports
func Run(t crossdock.T) {
	callBackHandler, serverCalledBack := newCallBackHandler()
	dispatcher, callBackAddr := dispatcher.CreateOnewayDispatcher(t, callBackHandler)
	defer dispatcher.Stop()

	encoding := t.Param(params.Encoding)
	switch encoding {
	case "raw":
		Raw(t, dispatcher, serverCalledBack, callBackAddr)
	case "json":
		JSON(t, dispatcher, serverCalledBack, callBackAddr)
	case "thrift":
		Thrift(t, dispatcher, serverCalledBack, callBackAddr)
	case "protobuf":
		// skip
	default:
		crossdock.Fatals(t).Fail("unknown encoding", "%v", encoding)
	}
}

// newCallBackHandler creates a oneway handler that fills a channel with the body
func newCallBackHandler() (raw.OnewayHandler, <-chan []byte) {
	serverCalledBack := make(chan []byte)
	return func(ctx context.Context, body []byte) error {
		serverCalledBack <- body
		return nil
	}, serverCalledBack
}

func getRandomID() string { return random.String(10) }
