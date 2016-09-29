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
	"fmt"
	"time"

	"context"

	"github.com/yarpc/yarpc-go"
)

// SleepRaw responds to raw requests over any transport by sleeping for one
// second.
func SleepRaw(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	time.Sleep(1 * time.Second)
	return nil, nil, nil
}

// Sleep responds to json requests over any transport by sleeping for one
// second.
func Sleep(ctx context.Context, reqMeta yarpc.ReqMeta, body interface{}) (interface{}, yarpc.ResMeta, error) {
	time.Sleep(1 * time.Second)
	return nil, nil, nil
}

// WaitForTimeoutRaw waits after the context deadline then returns the context
// error. yarpc should interpret this as an handler timeout, which in turns
// should be forwarded to the yarpc client as a remote handler timeout.
func WaitForTimeoutRaw(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, nil, fmt.Errorf("no deadline set in context")
	}
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}
