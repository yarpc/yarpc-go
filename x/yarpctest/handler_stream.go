// Copyright (c) 2019 Uber Technologies, Inc.
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
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/x/yarpctest/api"
)

// EchoStreamHandler is a Bidirectional Stream Handler that will echo any
// request that is sent to it.
func EchoStreamHandler() api.ProcOption {
	return &echoStreamHandler{}
}

type echoStreamHandler struct {
	api.SafeTestingTBOnStart

	wg      sync.WaitGroup
	stopped atomic.Bool
}

func (h *echoStreamHandler) ApplyProc(opts *api.ProcOpts) {
	opts.HandlerSpec = transport.NewStreamHandlerSpec(h)
}

func (h *echoStreamHandler) Stop(testing.TB) error {
	h.stopped.Store(true)
	h.wg.Wait()
	return nil
}

func (h *echoStreamHandler) HandleStream(s *transport.ServerStream) error {
	if h.stopped.Load() {
		return errors.New("closed")
	}
	h.wg.Add(1)
	defer h.wg.Done()
	for {
		msg, err := s.ReceiveMessage(context.Background())
		if err != nil {
			return err
		}
		err = s.SendMessage(context.Background(), msg)
		if err != nil {
			return err
		}
	}
}

// OrderedStreamHandler is a bidirectional stream handler that can apply stream
// actions in a specified order.
func OrderedStreamHandler(actions ...api.ServerStreamAction) api.ProcOption {
	return &orderedStreamHandler{
		actions: actions,
	}
}

type orderedStreamHandler struct {
	actions []api.ServerStreamAction

	wg      sync.WaitGroup
	stopped atomic.Bool
	t       testing.TB
}

// ApplyProc implements ProcOption.
func (o *orderedStreamHandler) ApplyProc(opts *api.ProcOpts) {
	opts.HandlerSpec = transport.NewStreamHandlerSpec(o)
}

// Start sets the TestingT to use for assertions.
func (o *orderedStreamHandler) Start(t testing.TB) error {
	o.t = t

	var err error
	for _, action := range o.actions {
		err = multierr.Append(err, action.Start(t))
	}
	return err
}

// Stop cleans up the handler, waiting until all streams have ended.
func (o *orderedStreamHandler) Stop(t testing.TB) error {
	var err error
	for _, action := range o.actions {
		err = multierr.Append(err, action.Stop(t))
	}
	o.stopped.Store(true)
	o.wg.Wait()
	return err
}

// HandleStream handles a stream request.
func (o *orderedStreamHandler) HandleStream(s *transport.ServerStream) error {
	if o.stopped.Load() {
		return errors.New("closed")
	}
	o.wg.Add(1)
	defer o.wg.Done()

	for i, action := range o.actions {
		if err := action.ApplyServerStream(s); err != nil {
			require.Equal(o.t, i+1, len(o.actions), "exited before all actions were run.")
			return err
		}
	}
	return nil
}

// SERVER-ONLY ACTIONS

// StreamHandlerError is an action to return an error from a ServerStream
// handler.
func StreamHandlerError(err error) api.ServerStreamAction {
	return api.ServerStreamActionFunc(func(c *transport.ServerStream) error {
		return err
	})
}
