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

package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"go.uber.org/yarpc/transport"
)

// CacheFilter is a filter
type CacheFilter interface {
	transport.Filter

	Invalidate()
}

type entry struct {
	Headers transport.Headers
	Body    []byte
}

type cacheFilter map[string]entry

// NewCacheFilter builds a new CacheFilter.
func NewCacheFilter() CacheFilter {
	cache := make(cacheFilter)
	return &cache
}

func (c *cacheFilter) Invalidate() {
	fmt.Println("invalidating")
	*c = make(cacheFilter)
}

func (c *cacheFilter) Call(ctx context.Context, request *transport.Request, out transport.Outbound) (*transport.Response, error) {
	data := *c

	// Read the entire request body to match against the cache
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	request.Body = ioutil.NopCloser(bytes.NewReader(body))

	if v, ok := data[string(body)]; ok {
		fmt.Println("cache hit")
		return &transport.Response{
			Headers: v.Headers,
			Body:    ioutil.NopCloser(bytes.NewReader(v.Body)),
		}, nil
	}

	fmt.Println("cache miss")
	res, err := out.Call(ctx, request)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	data[string(body)] = entry{Headers: res.Headers, Body: resBody}
	res.Body = ioutil.NopCloser(bytes.NewReader(resBody))
	return res, nil
}
