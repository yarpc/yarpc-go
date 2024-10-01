// Copyright (c) 2024 Uber Technologies, Inc.
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

package http2

import (
	"github.com/opentracing/opentracing-go"
	"go.uber.org/net/metrics"
	backoffapi "go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/backoff"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type transportOptions struct {
	keepAlive           time.Duration
	maxIdleConnsPerHost int
	idleConnTimeout     time.Duration
	disableKeepAlives   bool
	disableCompression  bool
	connTimeout         time.Duration
	connBackoffStrategy backoffapi.Strategy
	jitter              func(int64) int64
	buildClient         func(*transportOptions) *http.Client
	logger              *zap.Logger
	meter               *metrics.Scope
	tracer              opentracing.Tracer
	serviceName         string
}

var defaultTransportOptions = transportOptions{
	keepAlive:           defaultKeepAlive,
	maxIdleConnsPerHost: defaultMaxIdleConnsPerHost,
	connTimeout:         defaultConnTimeout,
	connBackoffStrategy: backoff.DefaultExponential,
	idleConnTimeout:     defaultIdleConnTimeout,
	jitter:              rand.Int63n,
	buildClient:         buildHTTPClient,
	tracer:              opentracing.GlobalTracer(),
}

// Transport keeps track of HTTP peers and the associated HTTP client. It
// allows using a single HTTP client to make requests to multiple YARPC
// services and pooling the resources needed therein.
type Transport struct {
	lock sync.Mutex
	once *lifecycle.Once

	client *http.Client

	connTimeout         time.Duration
	connBackoffStrategy backoffapi.Strategy
	innocenceWindow     time.Duration
	jitter              func(int64) int64

	tracer      opentracing.Tracer
	logger      *zap.Logger
	meter       *metrics.Scope
	serviceName string
}

var _ transport.Transport = (*Transport)(nil)

// NewTransport creates a new HTTP/2 transport with the given options.
func NewTransport(options *transportOptions) *Transport {
	if options == nil {
		options = &defaultTransportOptions
	}

	return &Transport{
		once:                lifecycle.NewOnce(),
		connTimeout:         options.connTimeout,
		connBackoffStrategy: options.connBackoffStrategy,
		innocenceWindow:     options.connTimeout,
		jitter:              options.jitter,
		tracer:              options.tracer,
		logger:              options.logger,
		meter:               options.meter,
		serviceName:         options.serviceName,
		client:              options.buildClient(options),
	}
}

func (t *Transport) Start() error {
	//TODO implement me
	panic("implement me")
}

func (t *Transport) Stop() error {
	//TODO implement me
	panic("implement me")
}

func (t *Transport) IsRunning() bool {
	//TODO implement me
	panic("implement me")
}

func buildHTTPClient(options *transportOptions) *http.Client {
	// TODO: add implementation
	return &http.Client{}
}
