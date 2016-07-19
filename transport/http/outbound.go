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

	"github.com/yarpc/yarpc-go/internal/baggage"
	"github.com/yarpc/yarpc-go/internal/errors"
	"github.com/yarpc/yarpc-go/transport"

	"github.com/uber-go/atomic"
	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

var (
	errOutboundAlreadyStarted = errors.ErrOutboundAlreadyStarted("http.Outbound")
	errOutboundNotStarted     = errors.ErrOutboundNotStarted("http.Outbound")
)

type outboundConfig struct {
	keepAlive time.Duration
}

var defaultConfig = outboundConfig{keepAlive: 30 * time.Second}

// OutboundOption customizes the behavior of an HTTP outbound.
type OutboundOption func(*outboundConfig)

// KeepAlive specifies the keep-alive period for the network connection. If
// zero, keep-alives are disabled.
//
// Defaults to 30 seconds.
func KeepAlive(t time.Duration) OutboundOption {
	return func(c *outboundConfig) {
		c.keepAlive = t
	}
}

// NewOutbound builds a new HTTP outbound that sends requests to the given
// URL.
func NewOutbound(url string, opts ...OutboundOption) transport.Outbound {
	cfg := defaultConfig
	for _, o := range opts {
		o(&cfg)
	}

	// Instead of using a global client for all outbounds, we use an HTTP
	// client per outbound if unspecified.
	client := buildClient(&cfg)

	// TODO: Use option pattern with varargs instead
	return outbound{Client: client, URL: url, started: atomic.NewBool(false)}
}

type outbound struct {
	started *atomic.Bool
	Client  *http.Client
	URL     string
}

func (o outbound) Start() error {
	if o.started.Swap(true) {
		return errOutboundAlreadyStarted
	}
	return nil
}

// Options for the HTTP transport.
func (outbound) Options() (o transport.Options) {
	return o
}

func (o outbound) Stop() error {
	if !o.started.Swap(false) {
		return errOutboundNotStarted
	}
	return nil
}

func (o outbound) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	if !o.started.Load() {
		// panic because there's no recovery from this
		panic(errOutboundNotStarted)
	}

	start := time.Now()
	deadline, _ := ctx.Deadline()
	ttl := deadline.Sub(start)

	request, err := http.NewRequest("POST", o.URL, req.Body)
	if err != nil {
		return nil, err
	}

	request.Header = applicationHeaders.ToHTTPHeaders(req.Headers, nil)
	if hs := baggage.FromContext(ctx); hs.Len() > 0 {
		request.Header = baggageHeaders.ToHTTPHeaders(hs, request.Header)
	}

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
			return nil, errors.NewTimeoutError(req.Service, req.Procedure, deadline.Sub(start))
		}

		return nil, err
	}

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		appHeaders := applicationHeaders.FromHTTPHeaders(
			response.Header, transport.NewHeaders())
		return &transport.Response{
			Headers: appHeaders,
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
		return nil, errors.RemoteBadRequestError(message)
	}

	return nil, errors.RemoteUnexpectedError(message)
}
