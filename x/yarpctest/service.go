// Copyright (c) 2017 Uber Technologies, Inc.
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
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally"
	"go.uber.org/multierr"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/x/yarpctest/api"
)

// HTTPService will create a runnable HTTP service.
func HTTPService(options ...api.ServiceOption) api.Lifecycle {
	opts := api.ServiceOpts{}
	for _, option := range options {
		option.ApplyService(&opts)
	}
	if opts.Listener != nil {
		if err := opts.Listener.Close(); err != nil {
			panic(err)
		}
	}
	inbound := http.NewTransport().NewInbound(fmt.Sprintf("127.0.0.1:%d", opts.Port))
	return createService(opts.Name, inbound, opts.Procedures, options)
}

// TChannelService will create a runnable TChannel service.
func TChannelService(options ...api.ServiceOption) api.Lifecycle {
	opts := api.ServiceOpts{}
	for _, option := range options {
		option.ApplyService(&opts)
	}
	if opts.Listener != nil {
		if err := opts.Listener.Close(); err != nil {
			panic(err)
		}
	}
	trans, err := tchannel.NewTransport(
		tchannel.ListenAddr(fmt.Sprintf("127.0.0.1:%d", opts.Port)),
		tchannel.ServiceName(opts.Name),
	)
	if err != nil {
		panic(err)
	}
	inbound := trans.NewInbound()
	return createService(opts.Name, inbound, opts.Procedures, options)
}

// GRPCService will create a runnable GRPC service.
func GRPCService(options ...api.ServiceOption) api.Lifecycle {
	opts := api.ServiceOpts{}
	for _, option := range options {
		option.ApplyService(&opts)
	}
	trans := grpc.NewTransport()
	listener := opts.Listener
	var err error
	if listener == nil {
		if listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", opts.Port)); err != nil {
			panic(err)
		}
	}
	inbound := trans.NewInbound(listener)
	return createService(opts.Name, inbound, opts.Procedures, options)
}

func createService(
	name string,
	inbound transport.Inbound,
	procedures []transport.Procedure,
	options []api.ServiceOption,
) *wrappedDispatcher {
	d := yarpc.NewDispatcher(
		yarpc.Config{
			Name:     name,
			Inbounds: yarpc.Inbounds{inbound},
			Metrics: yarpc.MetricsConfig{
				Tally: tally.NoopScope,
			},
		},
	)
	d.Register(procedures)
	return &wrappedDispatcher{
		Dispatcher: d,
		options:    options,
		procedures: procedures,
	}
}

type wrappedDispatcher struct {
	*yarpc.Dispatcher
	options    []api.ServiceOption
	procedures []transport.Procedure
}

func (w *wrappedDispatcher) Start(t testing.TB) error {
	var err error
	for _, option := range w.options {
		err = multierr.Append(err, option.Start(t))
	}
	for _, procedure := range w.procedures {
		if unary := procedure.HandlerSpec.Unary(); unary != nil {
			if lc, ok := unary.(api.Lifecycle); ok {
				err = multierr.Append(err, lc.Start(t))
			}
		}
		if oneway := procedure.HandlerSpec.Oneway(); oneway != nil {
			if lc, ok := oneway.(api.Lifecycle); ok {
				err = multierr.Append(err, lc.Start(t))
			}
		}
		if stream := procedure.HandlerSpec.Stream(); stream != nil {
			if lc, ok := stream.(api.Lifecycle); ok {
				err = multierr.Append(err, lc.Start(t))
			}
		}
	}
	err = multierr.Append(err, w.Dispatcher.Start())
	assert.NoError(t, err, "error starting dispatcher: %s", w.Name())
	return err
}

func (w *wrappedDispatcher) Stop(t testing.TB) error {
	var err error
	for _, option := range w.options {
		err = multierr.Append(err, option.Stop(t))
	}
	for _, procedure := range w.procedures {
		if unary := procedure.HandlerSpec.Unary(); unary != nil {
			if lc, ok := unary.(api.Lifecycle); ok {
				err = multierr.Append(err, lc.Stop(t))
			}
		}
		if oneway := procedure.HandlerSpec.Oneway(); oneway != nil {
			if lc, ok := oneway.(api.Lifecycle); ok {
				err = multierr.Append(err, lc.Stop(t))
			}
		}
		if stream := procedure.HandlerSpec.Stream(); stream != nil {
			if lc, ok := stream.(api.Lifecycle); ok {
				err = multierr.Append(err, lc.Stop(t))
			}
		}
	}
	err = multierr.Append(err, w.Dispatcher.Stop())
	assert.NoError(t, err, "error stopping dispatcher: %s", w.Name())
	return err
}
