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
	"go.uber.org/yarpc/internal/inboundmiddleware"
	"go.uber.org/yarpc/internal/outboundmiddleware"
	"go.uber.org/yarpc/transport"
)

// UnaryOutboundMiddlewares combines the given collection of unary outbound
// middlewares in-order into a single UnaryOutboundMiddleware.
func UnaryOutboundMiddlewares(middlewares ...transport.UnaryOutboundMiddleware) transport.UnaryOutboundMiddleware {
	return outboundmiddleware.UnaryChain(middlewares...)
}

// UnaryInboundMiddlewares combines the given collection of unary inbound
// middleware in-order into a single UnaryInboundMiddleware.
func UnaryInboundMiddlewares(middlewares ...transport.UnaryInboundMiddleware) transport.UnaryInboundMiddleware {
	return inboundmiddleware.UnaryChain(middlewares...)
}

// OnewayOutboundMiddlewares combines the given collection of unary outbound
// middlewares in-order into a single OnewayOutboundMiddleware.
func OnewayOutboundMiddlewares(middlewares ...transport.OnewayOutboundMiddleware) transport.OnewayOutboundMiddleware {
	return outboundmiddleware.OnewayChain(middlewares...)
}

// OnewayInboundMiddlewares combines the given collection of unary inbound
// middleware in-order into a single OnewayInboundMiddleware.
func OnewayInboundMiddlewares(middlewares ...transport.OnewayInboundMiddleware) transport.OnewayInboundMiddleware {
	return inboundmiddleware.OnewayChain(middlewares...)
}
