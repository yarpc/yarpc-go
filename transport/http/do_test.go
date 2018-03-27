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

package http

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/testtime"
)

func TestCallHttpSuccess(t *testing.T) {
	successServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()

			body, err := ioutil.ReadAll(req.Body)
			if assert.NoError(t, err) {
				assert.Equal(t, []byte("world"), body)
			}

			w.Header().Set("foo", "bar")
			_, err = w.Write([]byte("great success"))
			assert.NoError(t, err)
		},
	))
	defer successServer.Close()

	httpTransport := NewTransport()
	out := httpTransport.NewSingleOutbound(successServer.URL)
	require.NoError(t, out.Start(), "failed to start outbound")
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	req, err := http.NewRequest("GET", successServer.URL, bytes.NewReader([]byte("world")))
	require.NoError(t, err)
	req = req.WithContext(ctx)

	client := http.Client{Transport: out}

	res, err := client.Do(req)

	require.NoError(t, err)
	defer res.Body.Close()

	foo := res.Header.Get("foo")
	assert.NotNil(t, foo, "value for foo expected")
	assert.Equal(t, "bar", foo, "foo value mismatch")

	body, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("great success"), body)
}

func TestHTTPSuccessWithMethods(t *testing.T) {
	tests := []struct {
		desc   string
		status string
		method string
	}{
		{
			desc:   "GET success",
			status: "success",
			method: "GET",
		},
		{
			desc:   "POST success",
			status: "lolwut",
			method: "POST",
		},
		{
			desc:   "TRACE success",
			status: "lolwut",
			method: "TRACE",
		},
		{
			desc:   "PUT success",
			status: "lolwut",
			method: "PUT",
		},
	}

	httpTransport := NewTransport()

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.Header().Add("Status", tt.status)
					defer r.Body.Close()
				},
			))
			defer server.Close()

			out := httpTransport.NewSingleOutbound(server.URL)

			require.NoError(t, out.Start(), "failed to start outbound")
			defer out.Stop()

			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 100*testtime.Millisecond)
			defer cancel()

			req, err := http.NewRequest(tt.method, server.URL, bytes.NewReader([]byte("world")))
			require.NoError(t, err)
			req = req.WithContext(ctx)

			client := http.Client{Transport: out}

			res, err := client.Do(req)

			assert.Equal(t, res.Header.Get("Status"), tt.status, "%v: application status", tt.desc)
		})
	}
}

func TestHTTPErrorWithMethods(t *testing.T) {
	tests := []struct {
		desc   string
		status string
		method string
	}{
		{
			desc:   "GET error",
			status: "success",
			method: "GET",
		},
		{
			desc:   "POST error",
			status: "error",
			method: "POST",
		},
		{
			desc:   "TRACE error",
			status: "lolwut",
			method: "TRACE",
		},
		{
			desc:   "PUT error",
			status: "lolwut",
			method: "PUT",
		},
	}

	httpTransport := NewTransport()
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "Error out", http.StatusInternalServerError)
				},
			))
			defer server.Close()
			// starting outbound
			out := httpTransport.NewSingleOutbound(server.URL)

			require.NoError(t, out.Start(), "failed to start outbound")
			defer out.Stop()

			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 100*testtime.Millisecond)
			defer cancel()
			// creating request
			req, err := http.NewRequest(tt.method, server.URL, bytes.NewReader([]byte("world")))
			require.NoError(t, err)
			req = req.WithContext(ctx)

			client := http.Client{Transport: out}
			// making request
			res, err := client.Do(req)
			require.NoError(t, err)
			require.NoError(t, res.Body.Close(), "%v: failed to close response body")
		})
	}
}

func TestHTTPTimeout(t *testing.T) {
	tests := []struct {
		desc    string
		method  string
		path    string
		timeout time.Duration
	}{
		{
			desc:    "GET error",
			method:  "GET",
			path:    "/",
			timeout: 0,
		},
		{
			desc:    "POST error",
			method:  "POST",
			path:    "/",
			timeout: 0,
		},
	}

	httpTransport := NewTransport()

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "Error out", http.StatusInternalServerError)
				},
			))
			defer server.Close()

			out := httpTransport.NewSingleOutbound(server.URL)

			require.NoError(t, out.Start(), "failed to start outbound")
			defer out.Stop()

			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, tt.timeout)
			defer cancel()

			req, err := http.NewRequest(tt.method, server.URL, bytes.NewReader([]byte("world")))
			require.NoError(t, err)
			req = req.WithContext(ctx)

			client := http.Client{Transport: out}

			res, err := client.Do(req)

			assert.Error(t, err)
			assert.Nil(t, res)
		})
	}
}

func TestHTTPApplicationError(t *testing.T) {
	tests := []struct {
		desc   string
		status string
	}{
		{
			desc:   "ok",
			status: "success",
		},
		{
			desc:   "error",
			status: "error",
		},
		{
			desc:   "not an error",
			status: "lolwut",
		},
	}

	httpTransport := NewTransport()

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.Header().Add("Status", tt.status)
					defer r.Body.Close()
				},
			))
			defer server.Close()

			out := httpTransport.NewSingleOutbound(server.URL)

			require.NoError(t, out.Start(), "failed to start outbound")
			defer out.Stop()

			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 100*testtime.Millisecond)
			defer cancel()

			req, err := http.NewRequest("GET", server.URL, bytes.NewReader([]byte("world")))
			require.NoError(t, err)
			req = req.WithContext(ctx)

			client := http.Client{Transport: out}

			res, err := client.Do(req)

			assert.Equal(t, res.Header.Get("Status"), tt.status, "%v: application status", tt.desc)

			require.NoError(t, err)
			require.NoError(t, res.Body.Close())
		})
	}
}

func TestHTTPCallFailures(t *testing.T) {
	notFoundServer := httptest.NewServer(http.NotFoundHandler())
	defer notFoundServer.Close()

	internalErrorServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "great sadness", http.StatusInternalServerError)
		}))
	defer internalErrorServer.Close()

	httpTransport := NewTransport()

	tests := []struct {
		desc     string
		url      string
		messages []string
	}{
		{
			"not a URL",
			"not a URL",
			[]string{"protocol scheme"},
		},
		{
			"404",
			notFoundServer.URL,
			[]string{"404", "page not found"},
		},
		{
			"failed due to error",
			internalErrorServer.URL,
			[]string{"great sadness"},
		},
		{
			"fail for a fully qualified URL",
			"http://example.com/",
			[]string{"failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			out := httpTransport.NewSingleOutbound(tt.url)
			require.NoError(t, out.Start(), "failed to start outbound")
			defer out.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			defer cancel()
			req, _ := http.NewRequest("GET", internalErrorServer.URL, bytes.NewReader([]byte("world")))
			req = req.WithContext(ctx)

			client := http.Client{Transport: out}

			res, err := client.Do(req)
			assert.NoError(t, err)

			assert.NotEqual(t, res.Status, 200)
		})
	}
}
