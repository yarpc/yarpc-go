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

package json

import (
	"time"

	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

// ReqMeta is a JSON request without the body.
type ReqMeta struct {
	Context context.Context

	// TODO(abg): Expose service name

	// Name of the procedure being called.
	Procedure string

	// Request headers
	Headers transport.Headers

	// TTL is the amount of time in which this request is expected to finish.
	TTL time.Duration
}

// Note: The shape of this request object is extremely similar to the
// raw.ReqMeta object, but since we can't unify all the ReqMeta objects
// (thrift.ReqMeta is very different), each encoding will have its own ReqMeta
// object.
