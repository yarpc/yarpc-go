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

package yarpctest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/grpc"
	"go.uber.org/yarpc/x/yarpctest/api"
)

// GRPCStreamRequest creates a new grpc stream request.
func GRPCStreamRequest(options ...api.ClientStreamRequestOption) api.Action {
	opts := api.NewClientStreamRequestOpts()
	for _, option := range options {
		option.ApplyClientStreamRequest(&opts)
	}
	return api.ActionFunc(func(t testing.TB) {
		trans := grpc.NewTransport()
		out := trans.NewSingleOutbound(fmt.Sprintf("127.0.0.1:%d", opts.Port))

		require.NoError(t, trans.Start())
		defer func() { assert.NoError(t, trans.Stop()) }()

		require.NoError(t, out.Start())
		defer func() { assert.NoError(t, out.Stop()) }()

		err := callStream(t, out, opts.GiveRequest, opts.StreamActions)
		if len(opts.WantErrMsgs) > 0 {
			require.Error(t, err)
			for _, wantErrMsg := range opts.WantErrMsgs {
				require.Contains(t, err.Error(), wantErrMsg)
			}
			return
		}
		require.NoError(t, err)
	})
}

func callStream(
	t testing.TB,
	out transport.StreamOutbound,
	req *transport.StreamRequest,
	actions []api.ClientStreamAction,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	client, err := out.CallStream(ctx, req)
	if err != nil {
		return err
	}
	for _, a := range actions {
		a.ApplyClientStream(t, client)
	}
	return nil
}

// ClientStreamActions combines a series of client stream actions into actions
// that will be applied when the StreamRequest is run.
func ClientStreamActions(actions ...api.ClientStreamAction) api.ClientStreamRequestOption {
	return api.ClientStreamRequestOptionFunc(func(opts *api.ClientStreamRequestOpts) {
		opts.StreamActions = actions
	})
}

// WantStreamError asserts that the stream request had an error immediately.
func WantStreamError(wantErrMsgs ...string) api.ClientStreamRequestOption {
	return api.ClientStreamRequestOptionFunc(func(opts *api.ClientStreamRequestOpts) {
		opts.WantErrMsgs = wantErrMsgs
	})
}

// CLIENT-SPECIFIC STREAM ACTIONS (see stream.go for generic stream actions)

// CloseStream is an action to close a client stream.
func CloseStream() api.ClientStreamAction {
	return api.ClientStreamActionFunc(func(t testing.TB, c *transport.ClientStream) {
		require.NoError(t, c.Close(context.Background()))
	})
}
