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

package yarpchttp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
)

// parseTTL takes a context parses the given TTL, clamping the context to that
// TTL and as a side-effect, tracking any errors encountered while attempting
// to parse and validate that TTL.
//
// Leaves the context unchanged if the TTL is empty.
func parseTTL(ctx context.Context, req *yarpc.Request, ttl string) (_ context.Context, cancel func(), _ error) {
	if ttl == "" {
		return ctx, func() {}, nil
	}

	ttlms, err := strconv.Atoi(ttl)
	if err != nil {
		return ctx, func() {}, newInvalidTTLError(
			req.Service,
			req.Procedure,
			ttl,
		)
	}

	// negative TTLs are invalid
	if ttlms < 0 {
		return ctx, func() {}, newInvalidTTLError(
			req.Service,
			req.Procedure,
			fmt.Sprint(ttlms),
		)
	}

	ctx, cancel = context.WithTimeout(ctx, time.Duration(ttlms)*time.Millisecond)
	return ctx, cancel, nil
}

func newInvalidTTLError(service string, procedure string, ttl string) error {
	return yarpcerror.New(yarpcerror.CodeInvalidArgument, fmt.Sprintf("invalid TTL %q for service %q and procedure %q", ttl, service, procedure))
}
