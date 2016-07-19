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

	"golang.org/x/net/context"

	"github.com/yarpc/yarpc-go"
)

// SleepRaw responds to raw requests over any transport by sleeping for one
// second.
func SleepRaw(reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	time.Sleep(1 * time.Second)
	return nil, nil, nil
}

// TimeoutShortRaw timeouts after half the time of the remaning context
// deadline. This handler should fail with a Context timeout error, that yarpc
// should forward to the caller.
func TimeoutShortRaw(reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	ctx := reqMeta.Context()
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil, nil, fmt.Errorf("no deadline set in context")
	}
	timeout := (time.Now().Sub(deadline)) / 2
	ctx, _ = context.WithTimeout(ctx, timeout)
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}
