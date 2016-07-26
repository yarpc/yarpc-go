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

package errors

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/yarpc/yarpc-go/crossdock/client/params"

	"github.com/crossdock/crossdock-go"
)

type httpClient struct {
	c   *http.Client
	url *url.URL
}

type httpResponse struct {
	Body   string
	Status int
}

func (h httpClient) Call(t crossdock.T, hs map[string]string, body string) httpResponse {
	fatals := crossdock.Fatals(t)

	req := http.Request{
		Method:        "POST",
		URL:           h.url,
		ContentLength: int64(len(body)),
		Body:          ioutil.NopCloser(strings.NewReader(body)),
		Close:         true, // don't reuse connections
		Header:        make(http.Header),
	}
	for k, v := range hs {
		req.Header.Set(k, v)
	}

	res, err := h.c.Do(&req)
	fatals.NoError(err,
		"failed to make request(headers=%v, body=%q)", hs, body)

	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	fatals.NoError(err,
		"failed to read response for request(headers=%v, body=%q)", hs, body)

	return httpResponse{
		Body:   string(resBody),
		Status: res.StatusCode,
	}
}

func buildHTTPClient(t crossdock.T) httpClient {
	fatals := crossdock.Fatals(t)

	server := t.Param(params.Server)
	fatals.NotEmpty(server, "server is required")

	url, err := url.Parse(fmt.Sprintf("http://%s:8081", server))
	fatals.NoError(err, "failed to parse URL")

	return httpClient{
		c:   &http.Client{Transport: &http.Transport{}},
		url: url,
	}
}

// Run exercises a YARPC server with outbound HTTP requests from a rigged
// client and validates behavior that might only be visible to an HTTP client
// without the YARPC abstraction interposed, typically errors.
func Run(t crossdock.T) {
	client := buildHTTPClient(t)
	assert := crossdock.Assert(t)

	// one valid request before we throw the errors at it
	res := client.Call(t, map[string]string{
		"RPC-Caller":     "yarpc-test",
		"RPC-Service":    "yarpc-test",
		"RPC-Procedure":  "echo",
		"RPC-Encoding":   "json",
		"Context-TTL-MS": "100",
	}, `{"token":"10"}`)
	assert.Equal(200, res.Status,
		"valid request: should respond with status 200")

	// TODO: Uncomment. Currently failing with Node.
	// assert.Equal(`{"token":"10"}`+"\n", res.Body,
	//	"valid request: exact response body")

	assert.JSONEq(`{"token":"10"}`, res.Body,
		"valid request: response matches")

	tests := []struct {
		name    string
		headers map[string]string
		body    string

		wantStatus         int
		wantBody           string
		wantBodyStartsWith string
	}{
		{
			name:       "no service",
			headers:    map[string]string{},
			body:       "{}",
			wantStatus: 400,
			wantBody: "BadRequest: missing service name, procedure, " +
				"caller name, TTL, and encoding\n",
		},
		{
			name: "wrong service",
			headers: map[string]string{
				"RPC-Caller":     "yarpc-test",
				"RPC-Service":    "not-yarpc-test",
				"RPC-Procedure":  "echo",
				"RPC-Encoding":   "json",
				"Context-TTL-MS": "100",
			},
			body:       `{"token":"10"}`,
			wantStatus: 400,
			wantBody: `BadRequest: unrecognized procedure ` +
				`"echo" for service "not-yarpc-test"` + "\n",
		},
		{
			name: "no procedure",
			headers: map[string]string{
				"RPC-Service": "yarpc-test",
			},
			body:       "{}",
			wantStatus: 400,
			wantBody:   "BadRequest: missing procedure, caller name, TTL, and encoding\n",
		},
		{
			name: "no caller",
			headers: map[string]string{
				"RPC-Service":   "yarpc-test",
				"RPC-Procedure": "echo",
			},
			body:       "{}",
			wantStatus: 400,
			wantBody:   "BadRequest: missing caller name, TTL, and encoding\n",
		},
		{
			name: "no handler",
			headers: map[string]string{
				"RPC-Caller":     "yarpc-test",
				"RPC-Service":    "yarpc-test",
				"RPC-Procedure":  "no-such-procedure",
				"RPC-Encoding":   "json",
				"Context-TTL-MS": "100",
			},
			body:       "{}",
			wantStatus: 400,
			wantBody: `BadRequest: unrecognized procedure ` +
				`"no-such-procedure" for service "yarpc-test"` + "\n",
		},
		{
			name: "no timeout",
			headers: map[string]string{
				"RPC-Caller":    "yarpc-test",
				"RPC-Service":   "yarpc-test",
				"RPC-Procedure": "echo",
			},
			body:       "{}",
			wantStatus: 400,
			wantBody:   "BadRequest: missing TTL and encoding\n",
		},
		{
			name: "no encoding",
			headers: map[string]string{
				"RPC-Caller":     "yarpc-test",
				"RPC-Service":    "yarpc-test",
				"RPC-Procedure":  "echo",
				"Context-TTL-MS": "100",
			},
			body:       "{}",
			wantStatus: 400,
			wantBody:   "BadRequest: missing encoding\n",
		},
		{
			name: "invalid timeout",
			headers: map[string]string{
				"RPC-Caller":     "yarpc-test",
				"RPC-Service":    "yarpc-test",
				"RPC-Procedure":  "echo",
				"RPC-Encoding":   "json",
				"Context-TTL-MS": "moo",
			},
			body:       "{}",
			wantStatus: 400,
			wantBody: `BadRequest: invalid TTL "moo" for procedure "echo" ` +
				`of service "yarpc-test": must be positive integer` + "\n",
		},
		{
			name: "invalid request",
			headers: map[string]string{
				"RPC-Caller":     "yarpc-test",
				"RPC-Service":    "yarpc-test",
				"RPC-Procedure":  "echo",
				"RPC-Encoding":   "json",
				"Context-TTL-MS": "100",
			},
			body:       "i am not json",
			wantStatus: 400,
			wantBodyStartsWith: `BadRequest: failed to decode "json" request body ` +
				`for procedure "echo" of service "yarpc-test" from ` +
				`caller "yarpc-test":`,
		},
		{
			name: "encoding mismatch",
			headers: map[string]string{
				"RPC-Caller":     "yarpc-test",
				"RPC-Service":    "yarpc-test",
				"RPC-Procedure":  "echo",
				"RPC-Encoding":   "thrift",
				"Context-TTL-MS": "100",
			},
			body:       "{}",
			wantStatus: 400,
			wantBody: `BadRequest: failed to decode "json" request body ` +
				`for procedure "echo" of service "yarpc-test" from ` +
				`caller "yarpc-test": expected encoding "json" but got "thrift"` + "\n",
		},
		{
			name: "unexpected error",
			headers: map[string]string{
				"RPC-Caller":     "yarpc-test",
				"RPC-Service":    "yarpc-test",
				"RPC-Procedure":  "unexpected-error",
				"RPC-Encoding":   "json",
				"Context-TTL-MS": "100",
			},
			body:       "{}",
			wantStatus: 500,
			wantBody: `UnexpectedError: error for procedure "unexpected-error" ` +
				`of service "yarpc-test": error` + "\n",
		},
		{
			name: "bad response",
			headers: map[string]string{
				"RPC-Caller":     "yarpc-test",
				"RPC-Service":    "yarpc-test",
				"RPC-Procedure":  "bad-response",
				"RPC-Encoding":   "json",
				"Context-TTL-MS": "100",
			},
			body:       "{}",
			wantStatus: 500,
			wantBodyStartsWith: `UnexpectedError: failed to encode "json" response ` +
				`body for procedure "bad-response" of service "yarpc-test" ` +
				`from caller "yarpc-test":`,
		},
		{
			name: "remote bad request",
			headers: map[string]string{
				"RPC-Caller":     "yarpc-test",
				"RPC-Service":    "yarpc-test",
				"RPC-Procedure":  "phone",
				"RPC-Encoding":   "json",
				"Context-TTL-MS": "100",
			},
			body: `{
				"service": "yarpc-test",
				"procedure": "Echo::echo",
				"body": "not a Thrift payload",
				"transport": {"http": {"host": "` + t.Param(params.Server) + `", "port": 8081}}
			}`,
			wantStatus: 500,
			wantBodyStartsWith: `UnexpectedError: error for procedure "phone" of service "yarpc-test": ` +
				`BadRequest: failed to decode "thrift" request body for procedure "Echo::echo" ` +
				`of service "yarpc-test" from caller "yarpc-test": `,
		},
		{
			name: "remote unexpected error",
			headers: map[string]string{
				"RPC-Caller":     "yarpc-test",
				"RPC-Service":    "yarpc-test",
				"RPC-Procedure":  "phone",
				"RPC-Encoding":   "json",
				"Context-TTL-MS": "100",
			},
			body: `{
				"service": "yarpc-test",
				"procedure": "unexpected-error",
				"body": "{}",
				"transport": {"http": {"host": "` + t.Param(params.Server) + `", "port": 8081}}
			}`,
			wantStatus: 500,
			wantBodyStartsWith: `UnexpectedError: error for procedure "phone" of service "yarpc-test": ` +
				`UnexpectedError: error for procedure "unexpected-error" of service "yarpc-test": error` + "\n",
		},
	}

	for _, tt := range tests {
		res := client.Call(t, tt.headers, tt.body)
		t.Tag("scenario", tt.name)
		assert.Equal(tt.wantStatus, res.Status, "should respond with expected status")
		if tt.wantBody != "" {
			assert.Equal(tt.wantBody, res.Body, "response body should be informative error")
		}
		if tt.wantBodyStartsWith != "" {
			i := len(tt.wantBodyStartsWith)
			if i > len(res.Body) {
				i = len(res.Body)
			}
			body := res.Body[:i]
			assert.Equal(tt.wantBodyStartsWith, body,
				"%s: response body should be informative error", tt.name)
		}
	}
}
