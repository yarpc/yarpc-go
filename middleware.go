// Copyright (c) 2022 Uber Technologies, Inc.
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

// UnaryOutboundMiddleware combines the given collection of unary outbound
// middleware in-order into a single UnaryOutbound middleware.
func UnaryOutboundMiddleware(mw ...middleware.UnaryOutbound) middleware.UnaryOutbound {
	return outboundmiddleware.UnaryChain(mw...)
}

// UnaryInboundMiddleware combines the given collection of unary inbound
// middleware in-order into a single UnaryInbound middleware.
func UnaryInboundMiddleware(mw ...middleware.UnaryInbound) middleware.UnaryInbound {
	return inboundmiddleware.UnaryChain(mw...)
}

// OnewayOutboundMiddleware combines the given collection of unary outbound
// middleware in-order into a single OnewayOutbound middleware.
func OnewayOutboundMiddleware(mw ...middleware.OnewayOutbound) middleware.OnewayOutbound {
	return outboundmiddleware.OnewayChain(mw...)
}

// OnewayInboundMiddleware combines the given collection of unary inbound
// middleware in-order into a single OnewayInbound middleware.
func OnewayInboundMiddleware(mw ...middleware.OnewayInbound) middleware.OnewayInbound {
	return inboundmiddleware.OnewayChain(mw...)
}

// StreamOutboundMiddleware combines the given collection of unary outbound
// middleware in-order into a single StreamOutbound middleware.
func StreamOutboundMiddleware(mw ...middleware.StreamOutbound) middleware.StreamOutbound {
	return outboundmiddleware.StreamChain(mw...)
}

// StreamInboundMiddleware combines the given collection of unary inbound
// middleware in-order into a single StreamInbound middleware.
func StreamInboundMiddleware(mw ...middleware.StreamInbound) middleware.StreamInbound {
	return inboundmiddleware.StreamChain(mw...)
}
