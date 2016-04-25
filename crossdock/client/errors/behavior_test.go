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

package errors

import (
	"testing"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/client/behavior"
	"github.com/yarpc/yarpc-go/crossdock/server"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	rpc := yarpc.New(yarpc.Config{
		Name:     "yarpc-test",
		Inbounds: []transport.Inbound{http.NewInbound(":8081")},
	})

	json.Register(rpc, json.Procedure("echo", server.EchoJSON))
	json.Register(rpc, json.Procedure("unexpected-error", server.UnexpectedError))
	json.Register(rpc, json.Procedure("bad-response", server.BadResponse))

	require.NoError(t, rpc.Start(), "failed to start RPC server")
	defer rpc.Stop()

	params := behavior.ParamsFromMap{"server": "localhost"}
	entries := behavior.Run(func(s behavior.Sink) {
		Run(s, params)
	})

	for _, entry := range entries {
		e := entry.(behavior.Entry)
		assert.Equal(t, behavior.Passed, e.Status, e.Output)
	}
}
