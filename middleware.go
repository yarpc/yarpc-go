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

import (
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/internal/inboundmiddleware"
	"go.uber.org/yarpc/internal/outboundmiddleware"
)

// UnaryOutbound combines the given collection of unary outbound
// middleware in-order into a single UnaryOutbound.
func UnaryOutbound(mw ...middleware.UnaryOutbound) middleware.UnaryOutbound {
	return outboundmiddleware.UnaryChain(mw...)
}

// UnaryInbound combines the given collection of unary inbound
// middleware in-order into a single UnaryInbound.
func UnaryInbound(mw ...middleware.UnaryInbound) middleware.UnaryInbound {
	return inboundmiddleware.UnaryChain(mw...)
}

// OnewayOutbound combines the given collection of unary outbound
// middleware in-order into a single OnewayOutbound.
func OnewayOutbound(mw ...middleware.OnewayOutbound) middleware.OnewayOutbound {
	return outboundmiddleware.OnewayChain(mw...)
}

// OnewayInbound combines the given collection of unary inbound
// middleware in-order into a single OnewayInbound.
func OnewayInbound(mw ...middleware.OnewayInbound) middleware.OnewayInbound {
	return inboundmiddleware.OnewayChain(mw...)
}
