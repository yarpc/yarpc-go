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

package yarpctest

import "go.uber.org/yarpc"

// CallResMetaBuilder helps build fake yarpc.CallResMeta objects for testing.
//
// 	resMeta := NewCallResMetaBuilder(ctx).
// 		Headers(headers).
// 		Build()
type CallResMetaBuilder struct {
	singleUse

	headers yarpc.Headers
}

// NewCallResMetaBuilder builds a new fake ReqMeta for unit tests
func NewCallResMetaBuilder() *CallResMetaBuilder {
	return &CallResMetaBuilder{}
}

// Headers specifies the response headers for this CallResMeta.
func (f *CallResMetaBuilder) Headers(h yarpc.Headers) *CallResMetaBuilder {
	f.ensureUnused(f)
	f.headers = h
	return f
}

// Build builds a yarpc.ReqMeta from this CallResMetaBuilder.
//
// The CallResMetaBuilder is not re-usable and becomes invalid after this
// call.
func (f *CallResMetaBuilder) Build() yarpc.CallResMeta {
	resMeta := fakeResMeta(*f)
	f.use(f)
	return resMeta
}

type fakeResMeta CallResMetaBuilder

func (r fakeResMeta) Headers() yarpc.Headers {
	return CallResMetaBuilder(r).headers
}
