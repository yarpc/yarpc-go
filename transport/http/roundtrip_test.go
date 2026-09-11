// Copyright (c) 2026 Uber Technologies, Inc.
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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/yarpcerrors"
	"golang.org/x/net/http2"
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
	hreq := httptest.NewRequest("GET", echoServer.URL, strings.NewReader(giveBody))
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

// TestRoundTripHTTP2DoesNotAutoRedirect guards against a regression where
// routing HTTP/2 requests through the peer's dedicated *http.Client (instead
// of through a transportSender wrapping it) would make RoundTrip() follow
// redirects automatically via http.Client.Do, unlike the HTTP/1 path which
// calls Transport.RoundTrip directly and returns 3xx responses as-is.
func TestRoundTripHTTP2DoesNotAutoRedirect(t *testing.T) {
	redirectedTo := "/final"
	h2s := &http2.Server{IdleTimeout: defaultIdleConnTimeout}
	server := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == redirectedTo {
				w.Write([]byte("should not get here"))
				return
			}
			w.Header().Set("Location", redirectedTo)
			w.WriteHeader(http.StatusFound)
		},
	))
	server.Config.Protocols = new(http.Protocols)
	server.Config.Protocols.SetHTTP1(true)
	server.Config.Protocols.SetUnencryptedHTTP2(true)
	http2.ConfigureServer(server.Config, h2s)
	server.Start()
	defer server.Close()

	tran := NewTransport()
	defer tran.Stop()
	out := tran.NewSingleOutbound(server.URL, UseHTTP2())
	require.NoError(t, out.Start(), "failed to start outbound")
	defer out.Stop()

	req, err := http.NewRequest("GET", server.URL, nil /* body */)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()
	req = req.WithContext(ctx)

	res, err := out.RoundTrip(req)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusFound, res.StatusCode,
		"RoundTrip must return the 3xx response as-is, not follow the redirect")
	assert.Equal(t, redirectedTo, res.Header.Get("Location"))
}
