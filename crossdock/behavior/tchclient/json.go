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

package tchclient

import (
	"time"

	"github.com/yarpc/yarpc-go/crossdock-go"
	"github.com/yarpc/yarpc-go/crossdock/behavior/random"

	"github.com/uber/tchannel-go/json"
)

func runJSON(s crossdock.T, call call) {
	assert := crossdock.Assert(s)
	checks := crossdock.Checks(s)

	headers := map[string]string{
		"hello": "json",
	}
	token := random.String(5)

	resp, respHeaders, err := jsonCall(call, headers, token)
	if checks.NoError(err, "json: call failed") {
		assert.Equal(token, resp.Token, "body echoed")
		assert.Equal(headers, respHeaders, "headers echoed")
	}
}

type jsonResp struct {
	Token string `json:"token"`
}

func jsonCall(call call, headers map[string]string, token string) (jsonResp, map[string]string, error) {
	peer := call.Channel.Peers().Add(call.ServerHostPort)

	ctx, cancel := json.NewContext(time.Second)
	ctx = json.WithHeaders(ctx, headers)
	defer cancel()

	var response jsonResp
	err := json.CallPeer(ctx, peer, serverName, "echo", &jsonResp{Token: token}, &response)
	return response, ctx.ResponseHeaders(), err
}
