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

package timeout

import (
	"context"
	"strings"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/raw"
	disp "go.uber.org/yarpc/internal/crossdock/client/dispatcher"
	"go.uber.org/yarpc/internal/crossdock/client/params"
	"go.uber.org/yarpc/transport"

	"github.com/crossdock/crossdock-go"
)

// Run tests if a yarpc client returns correctly a client timeout error behind
// the TimeoutError interface when the context deadline is reached while the
// server is taking too long to respond.
func Run(t crossdock.T) {
	assert := crossdock.Assert(t)
	fatals := crossdock.Fatals(t)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	dispatcher := disp.Create(t)
	fatals.NoError(dispatcher.Start(), "could not start Dispatcher")
	defer dispatcher.Stop()

	ch := raw.New(dispatcher.ClientConfig("yarpc-test"))
	_, _, err := ch.Call(ctx, yarpc.NewReqMeta().Procedure("sleep/raw"), nil)
	fatals.Error(err, "expected a failure for timeout")

	if transport.IsBadRequestError(err) {
		t.Skipf("sleep/raw method not implemented: %v", err)
		return
	}

	assert.True(transport.IsTimeoutError(err), "returns a TimeoutError: %T", err)

	trans := t.Param(params.Transport)
	switch trans {
	case "http":
		form := strings.HasPrefix(err.Error(),
			`client timeout for procedure "sleep/raw" of service "yarpc-test" after`)
		assert.True(form, "should be a client timeout: %q", err.Error())
	case "tchannel":
		form := strings.HasPrefix(err.Error(), `timeout`)
		assert.True(form,
			"should be a remote timeout (we cant represent client timeout with tchannel): %q",
			err.Error())
	default:
		fatals.Fail("", "unknown transport %q", trans)
	}
}
