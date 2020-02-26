// Copyright (c) 2020 Uber Technologies, Inc.
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
	"time"

	"github.com/crossdock/crossdock-go"
	"go.uber.org/yarpc/encoding/raw"
	disp "go.uber.org/yarpc/internal/crossdock/client/dispatcher"
	"go.uber.org/yarpc/yarpcerrors"
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

	client := raw.New(dispatcher.ClientConfig("yarpc-test"))
	_, err := client.Call(ctx, "sleep/raw", nil)
	fatals.Error(err, "expected a failure for timeout")

	if yarpcerrors.FromError(err).Code() == yarpcerrors.CodeInvalidArgument {
		t.Skipf("sleep/raw method not implemented: %v", err)
		return
	}

	assert.Equal(yarpcerrors.CodeDeadlineExceeded, yarpcerrors.FromError(err).Code(), "is an error with code CodeDeadlineExceeded: %v", err)
}
