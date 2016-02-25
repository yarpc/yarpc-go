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

package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// start the inbound with a mock handler
	h := transporttest.NewMockHandler(mockCtrl)
	i := NewInbound("127.0.0.1:0")
	require.NoError(t, i.Start(h), "failed to Start()")
	defer i.Stop()

	request := &transport.Request{
		Caller:    "testclient",
		Service:   "mockservice",
		Procedure: "hello",
		Headers:   transport.Headers{"token": "1234"},
		Body:      bytes.NewReader([]byte("world")),
		TTL:       200 * time.Millisecond, // TODO use default
	}

	// Expect a call with the given request and write the respons.
	h.EXPECT().Handle(
		gomock.Any(), transporttest.NewRequestMatcher(t, request), gomock.Any(),
	).Do(
		func(_ context.Context, _ *transport.Request, rw transport.ResponseWriter) {
			rw.AddHeaders(transport.Headers{"status": "ok"})
			if _, err := rw.Write([]byte("hello, world")); err != nil {
				panic(fmt.Sprintf("unexpected error: %v", err))
			}
		},
	).Return(nil)

	o := NewOutbound(fmt.Sprintf("http://%v/", i.Addr().String()))
	res, err := o.Call(context.TODO(), request)

	if assert.NoError(t, err) {
		response := &transport.Response{
			Headers: transport.Headers{"status": "ok"},
			Body:    ioutil.NopCloser(bytes.NewReader([]byte("hello, world"))),
		}
		responseMatcher := transporttest.NewResponseMatcher(t, response)
		assert.True(t, responseMatcher.Matches(res))
	}
}
