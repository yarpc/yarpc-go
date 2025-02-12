// Copyright (c) 2025 Uber Technologies, Inc.
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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestRoundTripSuccess(t *testing.T) {
	headerKey, headerVal := "foo", "bar"
	giveBody := "successful response"

	echoServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()

			// copy header
			header := req.Header.Get(headerKey)
			w.Header().Set(headerKey, header)

			// copy body
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Error("error reading body")
			}
			_, err = w.Write(body)
			if err != nil {
				t.Error("error writing body")
			}
		},
	))
	defer echoServer.Close()

	// start outbound
	httpTransport := NewTransport()
	defer httpTransport.Stop()
	var out transport.UnaryOutbound = httpTransport.NewSingleOutbound(echoServer.URL)
	require.NoError(t, out.Start(), "failed to start outbound")
	defer out.Stop()

	// create request
	hreq := httptest.NewRequest("GET", echoServer.URL, bytes.NewReader([]byte(giveBody)))
	hreq.Header.Add(headerKey, headerVal)

	// add deadline
	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	hreq = hreq.WithContext(ctx)

	// make call
	rt, ok := out.(http.RoundTripper)
	assert.True(t, ok, "unable to convert an outbound to a http.RoundTripper")

	res, err := rt.RoundTrip(hreq)
	require.NoError(t, err, "could not make call")
	defer res.Body.Close()

	// validate header
	gotHeaderVal := res.Header.Get(headerKey)
	assert.Equal(t, headerVal, gotHeaderVal, "header did not match")

	// validate body
	gotBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, giveBody, string(gotBody), "body did not match")
}

func TestRoundTripTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done() // never respond
		}))
	defer server.Close()

	tran := NewTransport()
	defer tran.Stop()
	// start outbound
	out := tran.NewSingleOutbound(server.URL)
	require.NoError(t, out.Start(), "failed to start outbound")
	defer out.Stop()

	// create request
	req, err := http.NewRequest("POST", server.URL, nil /* body */)
	require.NoError(t, err)

	// set a small deadline so the the call times out quickly
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	// make call
	client := http.Client{Transport: out}
	res, err := client.Do(req)

	// validate response
	if assert.Error(t, err) {
		// we use a Contains here since the returned error is really a
		// url.Error wrapping a yarpcerror
		assert.Contains(t, err.Error(), yarpcerrors.CodeDeadlineExceeded.String())
	}
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	assert.Nil(t, res)
}

func TestRoundTripNoDeadline(t *testing.T) {
	URL := "http://foo-host"

	tran := NewTransport()
	defer tran.Stop()
	out := tran.NewSingleOutbound(URL)
	require.NoError(t, out.Start(), "could not start outbound")
	defer out.Stop()

	hreq, err := http.NewRequest("GET", URL, nil /* body */)
	require.NoError(t, err)

	resp, err := out.RoundTrip(hreq)
	assert.Equal(t, yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "missing context deadline"), err)
	assert.Nil(t, resp)
}

func TestRoundTripNotRunning(t *testing.T) {
	URL := "http://foo-host"
	out := NewTransport().NewSingleOutbound(URL)

	req, err := http.NewRequest("POST", URL, nil /* body */)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	client := http.Client{Transport: out}
	res, err := client.Do(req)

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "waiting for HTTP outbound to start")
	}
	assert.Nil(t, res)
}
