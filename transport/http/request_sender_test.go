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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// *http.Client do more than what a RoundTrip is supposed to do:
// - It tries to handle higher-level protocol details such as redirects, authentications, or cookies.
// - It requires a client request(requestURI can't be set) so it is not possible to proxy a server request transparently.
// We want to make sure transportSender can proxy server requests transparently.
func TestSender(t *testing.T) {
	const data = "dummy server response body"

	var (
		server = httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				io.WriteString(w, data)
			},
		))
		clientReq, _ = http.NewRequest("GET", server.URL, nil)
		serverReq    = httptest.NewRequest("GET", server.URL, nil)
		client       = &http.Client{
			Transport: http.DefaultTransport,
		}
	)
	defer server.Close()

	tests := []struct {
		msg            string
		sender         sender
		req            *http.Request
		wantStatusCode int
		wantBody       string
		wantError      string
	}{
		{
			msg:            "http.Client sender, http client request",
			req:            clientReq,
			sender:         http.DefaultClient,
			wantStatusCode: http.StatusOK,
			wantBody:       data,
		},
		{
			msg:       "http.Client sender, http server request",
			req:       serverReq,
			sender:    http.DefaultClient,
			wantError: "http: Request.RequestURI can't be set in client requests",
		},
		{
			msg:            "transportSender, http client request",
			req:            clientReq,
			sender:         &transportSender{Client: client},
			wantStatusCode: http.StatusOK,
			wantBody:       data,
		},
		{
			msg:            "transportSender, http server request",
			req:            serverReq,
			sender:         &transportSender{Client: client},
			wantStatusCode: http.StatusOK,
			wantBody:       data,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			resp, err := tt.sender.Do(tt.req)
			if tt.wantError != "" {
				require.Error(t, err, "expect error when we use http.Client to send a server request")
				assert.Contains(t, err.Error(), tt.wantError, "error body mismatch")
				return
			}
			assert.Equal(t, tt.wantStatusCode, resp.StatusCode, "status code does not match")
			body, _ := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			assert.Equal(t, tt.wantBody, string(body), "response body does not match")
		})
	}
}
