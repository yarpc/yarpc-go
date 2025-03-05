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

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
)

// CacheOutboundMiddleware is a OutboundMiddleware
type CacheOutboundMiddleware interface {
	middleware.UnaryOutbound

	Invalidate()
}

type entry struct {
	Headers transport.Headers
	Body    []byte
}

type cacheOutboundMiddleware map[string]entry

// NewCacheOutboundMiddleware builds a new CacheOutboundMiddleware.
func NewCacheOutboundMiddleware() CacheOutboundMiddleware {
	cache := make(cacheOutboundMiddleware)
	return &cache
}

func (c *cacheOutboundMiddleware) Invalidate() {
	fmt.Println("invalidating")
	*c = make(cacheOutboundMiddleware)
}

func (c *cacheOutboundMiddleware) Call(ctx context.Context, request *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	data := *c

	// Read the entire request body to match against the cache
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	request.Body = bytes.NewReader(body)

	if v, ok := data[string(body)]; ok {
		fmt.Println("cache hit")
		return &transport.Response{
			Headers: v.Headers,
			Body:    io.NopCloser(bytes.NewReader(v.Body)),
		}, nil
	}

	fmt.Println("cache miss")
	res, err := out.Call(ctx, request)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	data[string(body)] = entry{Headers: res.Headers, Body: resBody}
	res.Body = io.NopCloser(bytes.NewReader(resBody))
	return res, nil
}
