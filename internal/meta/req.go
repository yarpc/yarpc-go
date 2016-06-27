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

package meta

import (
	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

// FromTransportRequest builds a ReqMeta from a transport-level Request.
func FromTransportRequest(ctx context.Context, req *transport.Request) yarpc.ReqMeta {
	return reqMeta{ctx: ctx, req: req}
}

// ToTransportRequest fills the given transport request with information from
// the given ReqMeta. The Context associated with the ReqMeta is returned.
func ToTransportRequest(reqMeta yarpc.CallReqMeta, req *transport.Request) context.Context {
	req.Procedure = reqMeta.GetProcedure()
	req.Headers = transport.Headers(reqMeta.GetHeaders())
	return reqMeta.GetContext()
}

type reqMeta struct {
	ctx context.Context
	req *transport.Request
}

func (r reqMeta) Caller() string {
	return r.req.Caller
}

func (r reqMeta) Context() context.Context {
	return r.ctx
}

func (r reqMeta) Encoding() transport.Encoding {
	return r.req.Encoding
}

func (r reqMeta) Headers() yarpc.Headers {
	return yarpc.Headers(r.req.Headers)
}

func (r reqMeta) Procedure() string {
	return r.req.Procedure
}

func (r reqMeta) Service() string {
	return r.req.Service
}
