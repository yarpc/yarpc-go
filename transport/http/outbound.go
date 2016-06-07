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
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

// NewOutbound builds a new HTTP outbound that sends requests to the given
// URL.
func NewOutbound(url string) transport.Outbound {
	return NewOutboundWithClient(url, nil)
}

// NewOutboundWithClient builds a new HTTP outbound that sends requests to the
// given URL using the given HTTP client.
func NewOutboundWithClient(url string, client *http.Client) transport.Outbound {
	return outbound{Client: client, URL: url}
}

type outbound struct {
	Client *http.Client
	URL    string
}

func (o outbound) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	start := time.Now()
	deadline, _ := ctx.Deadline()
	ttl := deadline.Sub(start)

	request, err := http.NewRequest("POST", o.URL, req.Body)
	if err != nil {
		return nil, err
	}

	request.Header = applicationHeaders.ToHTTPHeaders(req.Headers, nil)
	request.Header.Set(CallerHeader, req.Caller)
	request.Header.Set(ServiceHeader, req.Service)
	request.Header.Set(ProcedureHeader, req.Procedure)
	request.Header.Set(TTLMSHeader, fmt.Sprintf("%d", ttl/time.Millisecond))

	encoding := string(req.Encoding)
	if encoding != "" {
		request.Header.Set(EncodingHeader, encoding)
	}

	response, err := ctxhttp.Do(ctx, o.Client, request)
	if err != nil {
		if err == context.DeadlineExceeded {
			return nil, transport.NewTimeoutError(req.Service, req.Procedure, deadline.Sub(start))
		}

		return nil, err
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return &transport.Response{
			Headers: applicationHeaders.FromHTTPHeaders(response.Header, nil),
			Body:    response.Body,
		}, nil
	}

	// TODO Behavior for 300-range status codes is undefined
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if err := response.Body.Close(); err != nil {
		return nil, err
	}

	// Trim the trailing newline from HTTP error messages
	message := strings.TrimSuffix(string(contents), "\n")

	if response.StatusCode >= 400 && response.StatusCode < 500 {
		return nil, transport.RemoteBadRequestError(message)
	}

	return nil, transport.RemoteUnexpectedError(message)
}
