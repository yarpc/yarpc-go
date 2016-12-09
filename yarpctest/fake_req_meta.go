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

import (
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
)

// ReqMetaBuilder helps build fake yarpc.ReqMeta objects for testing.
//
// 	reqMeta := NewReqMetaBuilder().
// 		Encoding(json.Encoding).
// 		Caller("mytestcaller").
// 		Procedure("hello").
// 		Service("testservice").
// 		Build()
type ReqMetaBuilder struct {
	singleUse

	encoding  transport.Encoding
	headers   yarpc.Headers
	caller    string
	procedure string
	service   string
}

// NewReqMetaBuilder instantiates a new ReqMetaBuilder with the given context.
func NewReqMetaBuilder() *ReqMetaBuilder {
	return &ReqMetaBuilder{}
}

// Encoding specifies the Encoding for the ReqMeta.
func (f *ReqMetaBuilder) Encoding(e transport.Encoding) *ReqMetaBuilder {
	f.ensureUnused(f)
	f.encoding = e
	return f
}

// Headers specifies the headers attached to this ReqMeta.
func (f *ReqMetaBuilder) Headers(h yarpc.Headers) *ReqMetaBuilder {
	f.ensureUnused(f)
	f.headers = h
	return f
}

// Caller specifies the name of the caller for this request.
func (f *ReqMetaBuilder) Caller(s string) *ReqMetaBuilder {
	f.caller = s
	return f
}

// Service specifies the target service name for this request.
func (f *ReqMetaBuilder) Service(s string) *ReqMetaBuilder {
	f.ensureUnused(f)
	f.service = s
	return f
}

// Procedure specifies the name of the procedure being called.
func (f *ReqMetaBuilder) Procedure(s string) *ReqMetaBuilder {
	f.ensureUnused(f)
	f.procedure = s
	return f
}

// Build builds a yarpc.ReqMeta from this ReqMetaBuilder.
//
// The ReqMetaBuilder is not re-usable and becomes invalid after this call.
func (f *ReqMetaBuilder) Build() yarpc.ReqMeta {
	reqMeta := fakeReqMeta(*f)
	f.use(f)
	return reqMeta
}

type fakeReqMeta ReqMetaBuilder

func (r fakeReqMeta) Caller() string {
	return ReqMetaBuilder(r).caller
}

func (r fakeReqMeta) Encoding() transport.Encoding {
	return ReqMetaBuilder(r).encoding
}

func (r fakeReqMeta) Headers() yarpc.Headers {
	return ReqMetaBuilder(r).headers
}

func (r fakeReqMeta) Procedure() string {
	return ReqMetaBuilder(r).procedure
}

func (r fakeReqMeta) Service() string {
	return ReqMetaBuilder(r).service
}
