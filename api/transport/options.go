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

package transport

import "go.uber.org/zap"

// Option allows customizing the invocation of YARPC handlers
type Option interface {
	invokerOption()
}

var _ Option = (*InvokerOption)(nil)

// InvokerOptions encapsulates customizations to invocation of handlers
type InvokerOptions struct {
	logger *zap.Logger
}

// InvokerOption customizes handler invocation
type InvokerOption func(*InvokerOptions)

func (InvokerOption) invokerOption() {}

// Logger sets optional zap logger
func Logger(l *zap.Logger) InvokerOption {
	return func(io *InvokerOptions) {
		io.logger = l
	}
}

// NewInvokerOptions returns struct of optional parameters to invoker
func NewInvokerOptions(options ...InvokerOption) *InvokerOptions {
	invokerOptions := &InvokerOptions{}
	for _, opt := range options {
		opt(invokerOptions)
	}
	return invokerOptions
}
