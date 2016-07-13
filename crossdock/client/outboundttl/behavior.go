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

package outboundttl

import (
	"strings"
	"time"

	"github.com/yarpc/yarpc-go"
	disp "github.com/yarpc/yarpc-go/crossdock/client/dispatcher"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/transport"

	"github.com/crossdock/crossdock-go"
	"golang.org/x/net/context"
)

// Run tests the behavior of any outbound transport to verify that it will
// timeout if the server fails to respond in a timely fashion.
func Run(t crossdock.T) {
	assert := crossdock.Assert(t)
	fatals := crossdock.Fatals(t)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	dispatcher := disp.Create(t)
	fatals.NoError(dispatcher.Start(), "could not start Dispatcher")
	defer dispatcher.Stop()

	ch := raw.New(dispatcher.Channel("yarpc-test"))
	_, _, err := ch.Call(yarpc.NewReqMeta(ctx).Procedure("sleep/raw"), nil)
	fatals.Error(err, "expected a failure for timeout")

	if transport.IsBadRequestError(err) {
		t.Skipf("sleep/raw method not implemented: %v", err)
		return
	}

	form := strings.HasPrefix(err.Error(), `timeout for procedure "sleep/raw" of service "yarpc-test" after`)
	assert.True(form, "error message has expected prefix for timeouts, got %q", err.Error())
	assert.True(
		transport.IsTimeoutError(err), "error should be a TimeoutError, got %T", err)
}
