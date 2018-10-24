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

package tchannel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/x/yarpctest"
	"go.uber.org/yarpc/x/yarpctest/api"
	"go.uber.org/yarpc/x/yarpctest/types"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestHandleResourceExhausted(t *testing.T) {
	serviceName := "test-service"
	procedureName := "test-procedure"
	port := uint16(8000)

	resourceExhaustedHandler := &types.UnaryHandler{
		Handler: api.UnaryHandlerFunc(func(context.Context, *transport.Request, transport.ResponseWriter) error {
			// eg: simulating a rate limiter that's reached its limit
			return yarpcerrors.Newf(yarpcerrors.CodeResourceExhausted, "resource exhausted: rate limit exceeded")
		})}

	service := yarpctest.TChannelService(
		yarpctest.Name(serviceName),
		yarpctest.Port(port),
		yarpctest.Proc(yarpctest.Name(procedureName), resourceExhaustedHandler),
	)
	require.NoError(t, service.Start(t))
	defer func() { require.NoError(t, service.Stop(t)) }()

	requests := yarpctest.ConcurrentAction(
		yarpctest.TChannelRequest(
			yarpctest.Service(serviceName),
			yarpctest.Port(port),
			yarpctest.Procedure(procedureName),
			yarpctest.GiveTimeout(time.Millisecond*100),

			// all TChannel requests should timeout and never actually receive
			// the resource exhausted error
			yarpctest.WantError("timeout"),
		),
		10,
	)
	requests.Run(t)
}
