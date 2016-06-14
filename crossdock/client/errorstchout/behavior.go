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

package errorstchout

import (
	"fmt"
	"time"

	"github.com/yarpc/yarpc-go/crossdock/client/params"

	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/json"
	"github.com/yarpc/yarpc-go/crossdock-go"
)

const (
	serverPort  = 8082
	serviceName = "yarpc-test"
)

type test struct {
	procedure string
	body      []byte
	validate  func(res3 []byte, isAppExpr bool, err error)
}

// Run exercises a YARPC server with outbound TChannel requests from a rigged
// client and validates behavior that might only be visible to a TChannel
// client without the YARPC abstraction interposed, typically errors.
func Run(t crossdock.T) {
	fatals := crossdock.Fatals(t)
	assert := crossdock.Fatals(t)

	tests := []test{
		{
			procedure: "echo",
			body:      []byte{},
			validate: func(res3 []byte, isAppErr bool, err error) {
				assert.False(isAppErr, "malformed body must not be application error")
				err, ok := err.(tchannel.SystemError)
				assert.True(ok, "malformed body must produce system error")
				if !ok {
					return
				}
				code := tchannel.GetSystemErrorCode(err)
				assert.Equal(tchannel.ErrCodeBadRequest, code, "must produce bad request error")
				assert.Contains(err.Error(), `failed to decode "json"`, "must mention failing to decode JSON in error message")
			},
		},
		{
			procedure: "",
			body:      []byte{},
			validate: func(res3 []byte, isAppErr bool, err error) {
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
			procedure: "no-such-procedure",
			body:      []byte{},
			validate: func(res3 []byte, isAppErr bool, err error) {
				assert.False(isAppErr, "no-such-procedure must not produce application error")
				err, ok := err.(tchannel.SystemError)
				assert.True(ok, "no-such-procedure must produce  system error")
				if !ok {
					return
				}
				code := tchannel.GetSystemErrorCode(err)
				assert.Equal(tchannel.ErrCodeBadRequest, code, "must produce bad request error")
				assert.Contains(err.Error(), `unrecognized procedure "no-such-procedure"`, "must mention unrecongized procedure in error message")
			},
		},
		{
			procedure: "bad-response",
			body:      []byte("{}"),
			validate: func(res3 []byte, isAppErr bool, err error) {
				assert.False(isAppErr, "bad-response must not produce an application error")
				err, ok := err.(tchannel.SystemError)
				assert.True(ok, "bad-response must produce system error")
				if !ok {
					return
				}
				code := tchannel.GetSystemErrorCode(err)
				assert.Equal(tchannel.ErrCodeUnexpected, code, "bad-response must produce unexpected error")
				assert.Contains(err.Error(), `failed to encode "json"`, "must mention failure to encode JSON in error message")
			},
		},
		{
			procedure: "unexpected-error",
			body:      []byte("{}"),
			validate: func(res3 []byte, isAppErr bool, err error) {
				assert.False(isAppErr, "unexpected-error procedure must not produce application error")
				err, ok := err.(tchannel.SystemError)
				assert.True(ok, "unexpected-error procedure must produce system error")
				code := tchannel.GetSystemErrorCode(err)
				assert.Equal(tchannel.ErrCodeUnexpected, code, "must produce transport error")
			},
		},
	}

	server := t.Param(params.Server)
	serverHostPort := fmt.Sprintf("%v:%v", server, serverPort)

	ch, err := tchannel.NewChannel(serviceName, nil)
	fatals.NoError(err, "Could not create channel")

	peer := ch.Peers().Add(serverHostPort)

	for _, tt := range tests {
		(func(tt test) {
			var req2, res2, res3 []byte
			var headers map[string]string

			ctx, cancel := json.NewContext(time.Second)
			defer cancel()

			ctx = json.WithHeaders(ctx, headers)

			as := "json"
			call, err := peer.BeginCall(ctx, serviceName, tt.procedure, &tchannel.CallOptions{Format: tchannel.Format(as)})
			fatals.NoError(err, "Could not begin call")

			err = tchannel.NewArgWriter(call.Arg2Writer()).Write(req2)
			fatals.NoError(err, "Could not write request headers")

			err = tchannel.NewArgWriter(call.Arg3Writer()).Write(tt.body)
			fatals.NoError(err, "Could not write request body")

			err = tchannel.NewArgReader(call.Response().Arg2Reader()).Read(&res2)
			if err != nil {
				tt.validate(res3, false, err)
				return
			}

			isAppErr := call.Response().ApplicationError()

			err = tchannel.NewArgReader(call.Response().Arg3Reader()).Read(&res3)
			if err != nil {
				tt.validate(res3, isAppErr, err)
				return
			}
		})(tt)
	}
}
