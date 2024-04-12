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

package yarpctest

import (
	"testing"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/x/yarpctest/api"
)

// Proc will create a new Procedure that can be included in a Service.
func Proc(options ...api.ProcOption) api.ServiceOption {
	return newProc(options...)
}

func newProc(options ...api.ProcOption) *proc {
	opts := api.ProcOpts{Name: "proc"}
	for _, option := range options {
		option.ApplyProc(&opts)
	}
	return &proc{
		procedure: transport.Procedure{
			Name:        opts.Name,
			HandlerSpec: opts.HandlerSpec,
		},
		options: options,
	}
}

type proc struct {
	procedure transport.Procedure
	options   []api.ProcOption
}

// ApplyService implements ServiceOption.
func (p *proc) ApplyService(opts *api.ServiceOpts) {
	opts.Procedures = append(opts.Procedures, p.procedure)
}

// Start implements Lifecycle.
func (p *proc) Start(t testing.TB) error {
	var err error
	for _, option := range p.options {
		err = multierr.Append(err, option.Start(t))
	}
	return err
}

// Stop implements Lifecycle.
func (p *proc) Stop(t testing.TB) error {
	var err error
	for _, option := range p.options {
		err = multierr.Append(err, option.Stop(t))
	}
	return err
}
