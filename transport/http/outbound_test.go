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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/transport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestCallSuccess(t *testing.T) {
	successServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()

			assert.Equal(t, "caller", req.Header.Get(CallerHeader))
			assert.Equal(t, "service", req.Header.Get(ServiceHeader))
			assert.Equal(t, "raw", req.Header.Get(EncodingHeader))
			assert.Equal(t, "1000", req.Header.Get(TTLMSHeader))
			assert.Equal(t, "hello", req.Header.Get(ProcedureHeader))

			body, err := ioutil.ReadAll(req.Body)
			if assert.NoError(t, err) {
				assert.Equal(t, []byte("world"), body)
			}

			w.Header().Set("rpc-header-foo", "bar")
			_, err = w.Write([]byte("great success"))
			assert.NoError(t, err)
		},
	))
	defer successServer.Close()

	out := NewOutbound(successServer.URL)
	res, err := out.Call(context.TODO(), &transport.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  raw.Encoding,
		TTL:       time.Second,
		Procedure: "hello",
		Body:      bytes.NewReader([]byte("world")),
	})
	require.NoError(t, err)
	defer res.Body.Close()

	foo, ok := res.Headers.Get("foo")
	assert.True(t, ok, "value for foo expected")
	assert.Equal(t, "bar", foo, "foo value mismatch")

	body, err := ioutil.ReadAll(res.Body)
	if assert.NoError(t, err) {
		assert.Equal(t, []byte("great success"), body)
	}
}

func TestCallFailures(t *testing.T) {
	notFoundServer := httptest.NewServer(http.NotFoundHandler())
	defer notFoundServer.Close()

	internalErrorServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "great sadness", http.StatusInternalServerError)
		}))
	defer internalErrorServer.Close()

	tests := []struct {
		url      string
		messages []string
	}{
		{"not a URL", []string{"unsupported protocol scheme"}},
		{notFoundServer.URL, []string{"404", "page not found"}},
		{internalErrorServer.URL, []string{"great sadness"}},
	}

	for _, tt := range tests {
		out := NewOutboundWithClient(tt.url, http.DefaultClient)
		_, err := out.Call(context.TODO(), &transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  raw.Encoding,
			TTL:       time.Second,
			Procedure: "wat",
			Body:      bytes.NewReader([]byte("huh")),
		})
		assert.Error(t, err, "expected failure")
		for _, msg := range tt.messages {
			assert.Contains(t, err.Error(), msg)
		}
	}
}
