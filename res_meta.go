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

import "golang.org/x/net/context"

// ResMeta contains information about an outgoing YARPC response.
type ResMeta interface {
	Headers(Headers) ResMeta

	GetContext() context.Context
	GetHeaders() Headers
}

// CallResMeta contains information about an incoming YARPC response.
type CallResMeta interface {
	Context() context.Context
	Headers() Headers
}

// NewResMeta constructs a ResMeta with the given Context.
//
// The context MUST NOT be nil.
func NewResMeta(ctx context.Context) ResMeta {
	if ctx == nil {
		panic("invalid usage of ResMeta: context cannot be nil")
	}
	return &resMeta{ctx: ctx}
}

type resMeta struct {
	ctx     context.Context
	headers Headers
}

func (r *resMeta) Headers(h Headers) ResMeta {
	r.headers = h
	return r
}

func (r *resMeta) GetContext() context.Context {
	return r.ctx
}

func (r *resMeta) GetHeaders() Headers {
	return r.headers
}
