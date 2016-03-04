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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/transport"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

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
		{internalErrorServer.URL, []string{"500", "great sadness"}},
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
