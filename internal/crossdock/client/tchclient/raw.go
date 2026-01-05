// Copyright (c) 2026 Uber Technologies, Inc.
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

package tchclient

import (
	"context"
	"time"

	"github.com/crossdock/crossdock-go"
	"github.com/uber/tchannel-go/raw"
	"go.uber.org/yarpc/internal/crossdock/client/random"
)

func runRaw(t crossdock.T, call call) {
	assert := crossdock.Assert(t)
	checks := crossdock.Checks(t)

	headers := []byte{
		0x00, 0x01, // 1 header
		0x00, 0x05, // length = 5
		'h', 'e', 'l', 'l', 'o',
		0x00, 0x03, // length = 3
		'r', 'a', 'w',
	}
	expectedHeaderContains := []byte{
		0x00, 0x05, // length = 5
		'h', 'e', 'l', 'l', 'o',
		0x00, 0x03, // length = 3
		'r', 'a', 'w',
	}
	token := random.Bytes(5)

	resp, respHeaders, err := rawCall(call, headers, token)
	if checks.NoError(err, "raw: call failed") {
		assert.Equal(token, resp, "body echoed")
		assert.Contains(
			string(respHeaders),
			string(expectedHeaderContains),
			"headers echoed",
		)
	}
}

func rawCall(call call, headers []byte, token []byte) ([]byte, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	arg2, arg3, _, err := raw.Call(
		ctx,
		call.Channel,
		call.ServerHostPort,
		serverName,
		"echo/raw",
		headers,
		token,
	)
	return arg3, arg2, err
}
