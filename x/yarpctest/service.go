// Copyright (c) 2026 Uber Technologies, Inc.
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
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	return startThatCreatesStopFunc(func(t testing.TB) (stopper func(testing.TB) error, startErr error) {
		opts := api.ServiceOpts{}
		for _, option := range options {
			option.ApplyService(&opts)
		}
		if opts.Listener != nil {
			require.NoError(t, opts.Listener.Close())
		}
		inbound := http.NewTransport().NewInbound(fmt.Sprintf("127.0.0.1:%d", opts.Port))
		s := createService(opts.Name, inbound, opts.Procedures, options)
		return s.Stop, s.Start(t)
	})
}

// TChannelService will create a runnable TChannel service.
func TChannelService(options ...api.ServiceOption) api.Lifecycle {
	return startThatCreatesStopFunc(func(t testing.TB) (stopper func(testing.TB) error, startErr error) {
		opts := api.ServiceOpts{}
		for _, option := range options {
			option.ApplyService(&opts)
		}
		listener := opts.Listener
		var err error
		if listener == nil {
			listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", opts.Port))
			require.NoError(t, err)
		}
		trans, err := tchannel.NewTransport(
			tchannel.ServiceName(opts.Name),
			tchannel.Listener(listener),
		)
		require.NoError(t, err)
		inbound := trans.NewInbound()
		s := createService(opts.Name, inbound, opts.Procedures, options)
		return s.Stop, s.Start(t)
	})
}

// GRPCService will create a runnable GRPC service.
func GRPCService(options ...api.ServiceOption) api.Lifecycle {
	return startThatCreatesStopFunc(func(t testing.TB) (stopper func(testing.TB) error, startErr error) {
		opts := api.ServiceOpts{}
		for _, option := range options {
			option.ApplyService(&opts)
		}
		trans := grpc.NewTransport()
		listener := opts.Listener
		var err error
		if listener == nil {
			listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", opts.Port))
			require.NoError(t, err)
		}
		inbound := trans.NewInbound(listener)
		service := createService(opts.Name, inbound, opts.Procedures, options)
		return service.Stop, service.Start(t)
	})
}

func startThatCreatesStopFunc(startToStop func(t testing.TB) (stopper func(testing.TB) error, startErr error)) api.Lifecycle {
	return &startToStopper{
		startWithReturnedStop: startToStop,
	}
}

type startToStopper struct {
	startWithReturnedStop func(testing.TB) (stopper func(testing.TB) error, startErr error)
	stop                  func(testing.TB) error
}

func (s *startToStopper) Start(t testing.TB) error {
	var err error
	s.stop, err = s.startWithReturnedStop(t)
	return err
}

func (s *startToStopper) Stop(t testing.TB) error {
	if s.stop == nil {
		return errors.New("did not start lifecycle")
	}
	return s.stop(t)
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
