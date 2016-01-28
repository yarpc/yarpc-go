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
	"net/http"

	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

// Outbound TODO
func Outbound(url string) transport.Outbound {
	return httpOutbound{URL: url}
}

// OutboundWithClient TODO
func OutboundWithClient(url string, client *http.Client) transport.Outbound {
	return httpOutbound{Client: client, URL: url}
}

type httpOutbound struct {
	*http.Client

	URL string
}

func (h httpOutbound) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	request, err := http.NewRequest("POST", h.URL, req.Body)
	if err != nil {
		return nil, err
	}

	headers := make(map[string][]string, len(req.Headers))
	for k, v := range req.Headers {
		headers[k] = []string{v}
	}
	request.Header = headers

	_, err = ctxhttp.Do(ctx, h.Client, request)
	if err != nil {
		return nil, err
	}

	// TODO
	return nil, nil
}
