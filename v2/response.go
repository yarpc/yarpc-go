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

package yarpc

import "go.uber.org/yarpc/v2/yarpcerror"

// Response is the low level response representation.
type Response struct {
	// Peer is the address of the peer that handled the request, if known.
	//
	// Depending on the application, the peer that handle the request might
	// be preferred for a follow-up request, though it is generally better
	// to use a sharded peer chooser so retries go to an available peer if this
	// peer is no longer available.
	Peer Identifier

	// Headers are response headers or trailers.
	Headers Headers

	// ApplicationErrorInfo indicates that the response body contains a payload
	// that represents an error in the request encoding.
	ApplicationErrorInfo *yarpcerror.Info
}
