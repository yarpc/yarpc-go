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

package httpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	yarpc "go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/crossdock/client/params"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport"
	ht "go.uber.org/yarpc/transport/http"

	crossdock "github.com/crossdock/crossdock-go"
)

// Run exercise a yarpc client against a rigged httpserver.
func Run(t crossdock.T) {
	fatals := crossdock.Fatals(t)

	server := t.Param(params.HTTPServer)
	fatals.NotEmpty(server, "server is required")

	disp := yarpc.NewDispatcher(yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"yarpc-test": {
				Unary: ht.NewOutbound(fmt.Sprintf("http://%s:8085", server)),
			},
		},
	})

	fatals.NoError(disp.Start(), "could not start Dispatcher")
	defer disp.Stop()

	runRaw(t, disp)
}

// runRaw tests if a yarpc client returns a remote timeout error behind the
// TimeoutError interface when a remote http handler returns a handler timeout.
func runRaw(t crossdock.T, disp yarpc.Dispatcher) {
	assert := crossdock.Assert(t)
	fatals := crossdock.Fatals(t)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch := raw.New(disp.Channel("yarpc-test"))
	_, _, err := ch.Call(ctx, yarpc.NewReqMeta().Procedure("handlertimeout/raw"), nil)
	fatals.Error(err, "expected an error")

	if transport.IsBadRequestError(err) {
		t.Skipf("handlertimeout/raw method not implemented: %v", err)
		return
	}

	assert.True(transport.IsTimeoutError(err), "returns a TimeoutError: %T", err)

	form := strings.HasPrefix(err.Error(),
		`Timeout: call to procedure "handlertimeout/raw" of service "service" from caller "caller" timed out after`)
	assert.True(form, "must be a remote handler timeout: %q", err.Error())
}
