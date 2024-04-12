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

package debug

import "go.uber.org/zap"

// Option is an interface for customizing debug handlers.
type Option interface {
	apply(*options)
}

type optionFunc func(*options)

// opts represents the combined options supplied by the user.
type options struct {
	logger *zap.Logger
	tmpl   templateIface
}

// Logger specifies the logger that should be used to log.
// Default value is noop zap logger.
func Logger(logger *zap.Logger) Option {
	return optionFunc(func(opts *options) {
		opts.logger = logger
	})
}

// tmpl specifies the template to use.
// It is only used for testing.
func tmpl(tmpl templateIface) Option {
	return optionFunc(func(opts *options) {
		opts.tmpl = tmpl
	})
}
func (f optionFunc) apply(options *options) { f(options) }

// applyOptions creates new opts based on the given options.
func applyOptions(opts ...Option) options {
	options := options{
		logger: zap.NewNop(),
		tmpl:   _defaultTmpl,
	}
	for _, opt := range opts {
		opt.apply(&options)
	}
	return options
}
