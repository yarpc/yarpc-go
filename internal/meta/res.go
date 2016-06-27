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

// FromTransportResponse builds a CallResMeta from a transport-level Response.
func FromTransportResponse(ctx context.Context, res *transport.Response) yarpc.CallResMeta {
	return callResMeta{ctx: ctx, res: res}
}

// ToTransportResponseWriter fills the given transport response with
// information from the given ResMeta. The Context associated with the ResMeta
// is returned.
func ToTransportResponseWriter(resMeta yarpc.ResMeta, w transport.ResponseWriter) context.Context {
	if hs := resMeta.GetHeaders(); hs.Len() > 0 {
		w.AddHeaders(transport.Headers(resMeta.GetHeaders()))
	}
	return resMeta.GetContext()
}

type callResMeta struct {
	ctx context.Context
	res *transport.Response
}

func (r callResMeta) Context() context.Context {
	return r.ctx
}

func (r callResMeta) Headers() yarpc.Headers {
	return yarpc.Headers(r.res.Headers)
}
