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

package errorstchclient

import (
	"fmt"
	"time"

	"github.com/crossdock/crossdock-go"
	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/json"
	"go.uber.org/yarpc/internal/crossdock/client/params"
)

const (
	serverPort  = 8082
	serviceName = "yarpc-test"
)

type test struct {
	name      string
	encoding  string
	procedure string
	headers   []byte
	body      []byte
	validate  func(res3 []byte, isAppExpr bool, err error)
}

// Run exercises a YARPC server with outbound TChannel requests from a rigged
// client and validates behavior that might only be visible to a TChannel
// client without the YARPC abstraction interposed, typically errors.
func Run(t crossdock.T) {
	fatals := crossdock.Fatals(t)
	assert := crossdock.Assert(t)

	tests := []test{
		{
			name:      "happy path",
			procedure: "echo",
			body:      []byte("{}"),
			headers:   []byte("{}"),
			validate: func(res3 []byte, isAppErr bool, err error) {
				assert.NoError(err, "is not error")
				assert.False(isAppErr, "malformed body must not be application error")
			},
		},
		{
			name:      "malformed body",
			procedure: "echo",
			body:      []byte(""),
			headers:   []byte("{}"),
			validate: func(res3 []byte, isAppErr bool, err error) {
				assert.Error(err, "is error")
				assert.False(isAppErr, "malformed body must not be application error")
				err, ok := err.(tchannel.SystemError)
				assert.True(ok, "malformed body must produce system error")
				if !ok {
					return
				}
				code := tchannel.GetSystemErrorCode(err)
				assert.Contains(err.Error(), `failed to decode "json"`, "must mention failing to decode JSON in error message")
				assert.Equal(tchannel.ErrCodeBadRequest, code, "must produce bad request error")
			},
		},
		// TODO test invalid headers
		{
			name:      "missing procedure",
			procedure: "",
			body:      []byte{},
			headers:   []byte("{}"),
			validate: func(res3 []byte, isAppErr bool, err error) {
				assert.Error(err, "is error")
				assert.False(isAppErr, "missing procedure must not produce an application error")
				err, ok := err.(tchannel.SystemError)
				assert.True(ok, "missing procedure must produce system error")
				if !ok {
					return
				}
				code := tchannel.GetSystemErrorCode(err)
				assert.Equal(tchannel.ErrCodeBadRequest, code, "missing procedure must produce bad request error")
				assert.Contains(err.Error(), "missing procedure", "must mention missing procedure in error message")
			},
		},
		{
			name:      "invalid procedure",
			procedure: "no-such-procedure",
			body:      []byte{},
			headers:   []byte("{}"),
			validate: func(res3 []byte, isAppErr bool, err error) {
				assert.Error(err, "is error")
				assert.False(isAppErr, "no-such-procedure must not produce application error")
				err, ok := err.(tchannel.SystemError)
				assert.True(ok, "no-such-procedure must produce  system error")
				if !ok {
					return
				}
				code := tchannel.GetSystemErrorCode(err)
				assert.Equal(tchannel.ErrCodeBadRequest, code, "must produce bad request error: %v", err)
				assert.Contains(err.Error(), `unrecognized procedure "no-such-procedure"`, "must mention unrecongized procedure in error message")
			},
		},
		{
			name:      "bad response",
			procedure: "bad-response",
			body:      []byte("{}"),
			headers:   []byte("{}"),
			validate: func(res3 []byte, isAppErr bool, err error) {
				assert.Error(err, "is error")
				assert.False(isAppErr, "bad-response must not produce an application error")
				err, ok := err.(tchannel.SystemError)
				assert.True(ok, "bad-response must produce system error")
				if !ok {
					return
				}
				code := tchannel.GetSystemErrorCode(err)
				assert.Equal(tchannel.ErrCodeBadRequest, code, "bad-response must produce unexpected error")
				assert.Contains(err.Error(), `failed to encode "json"`, "must mention failure to encode JSON in error message")
			},
		},
		{
			name:      "unexpected error",
			procedure: "unexpected-error",
			body:      []byte("{}"),
			headers:   []byte("{}"),
			validate: func(res3 []byte, isAppErr bool, err error) {
				assert.NoError(err, "not error")
				assert.True(isAppErr, "unexpected-error procedure must produce application error")
			},
		},
	}

	server := t.Param(params.Server)
	serverHostPort := fmt.Sprintf("%v:%v", server, serverPort)

	ch, err := tchannel.NewChannel(serviceName, nil)
	fatals.NoError(err, "could not create channel")

	peer := ch.Peers().Add(serverHostPort)

	for _, tt := range tests {
		var res2, res3 []byte
		var headers map[string]string

		t.Tag("case", tt.name)
		t.Tag("procedure", tt.procedure)

		ctx, cancel := json.NewContext(time.Second)
		defer cancel()

		ctx = json.WithHeaders(ctx, headers)

		encoding := "json"
		if tt.encoding != "" {
			encoding = tt.encoding
		}

		call, err := peer.BeginCall(
			ctx,
			serviceName,
			tt.procedure,
			&tchannel.CallOptions{Format: tchannel.Format(encoding)},
		)
		fatals.NoError(err, "could not begin call")

		err = tchannel.NewArgWriter(call.Arg2Writer()).Write(tt.headers)
		fatals.NoError(err, "could not write request headers")

		err = tchannel.NewArgWriter(call.Arg3Writer()).Write(tt.body)
		fatals.NoError(err, "could not write request body")

		err = tchannel.NewArgReader(call.Response().Arg2Reader()).Read(&res2)
		isAppErr := call.Response().ApplicationError()
		if err == nil {
			err = tchannel.NewArgReader(call.Response().Arg3Reader()).Read(&res3)
		}

		tt.validate(res3, isAppErr, err)
	}
}
