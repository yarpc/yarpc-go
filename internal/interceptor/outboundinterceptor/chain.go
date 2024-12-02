// Copyright (c) 2024 Uber Technologies, Inc.
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

package outboundinterceptor

import (
	"go.uber.org/yarpc/internal/interceptor"
)

// UnaryChain combines a series of `UnaryOutbound`s into a single `UnaryOutbound`.
func UnaryChain(mw ...interceptor.UnaryOutbound) interceptor.UnaryOutbound {
	// TODO: implement
	return interceptor.NopUnaryOutbound
}

// OnewayChain combines a series of `UnaryOutbound`s into a single `UnaryOutbound`.
func OnewayChain(mw ...interceptor.OnewayOutbound) interceptor.OnewayOutbound {
	// TODO: implement
	return interceptor.NopOnewayOutbound
}

// StreamChain combines a series of `UnaryOutbound`s into a single `UnaryOutbound`.
func StreamChain(mw ...interceptor.StreamOutbound) interceptor.StreamOutbound {
	// TODO: implement
	return interceptor.NopStreamOutbound
}
