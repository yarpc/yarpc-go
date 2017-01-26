// Copyright (c) 2017 Uber Technologies, Inc.
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

package cherami

import (
	"fmt"

	"go.uber.org/yarpc/api/transport"
	intsync "go.uber.org/yarpc/internal/sync"
	"go.uber.org/yarpc/transport/x/cherami/internal"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/cherami-client-go/client/cherami"
)

// TransportConfig defines the config in order to create a cherami transport
// ServiceName should be the name of the service that is using yarpc
// if HyperbahnHostFile is provided, hyperbahn will be used to connect to cherami
// Otherwise, the provided frontend IP and port will be used to connect to cherami
type TransportConfig struct {
	ServiceName       string
	HyperbahnHostFile string
	FrontendIP        string
	Port              int
}

// NewTransport creates a new cherami transport for shared objects between inbound and outbound
func NewTransport(config TransportConfig) *Transport {
	return &Transport{
		config:        config,
		tracer:        opentracing.GlobalTracer(),
		clientFactory: internal.NewClientFactory(),
	}
}

// Transport keeps shared objects between inbound and outbound
type Transport struct {
	once intsync.LifecycleOnce

	config        TransportConfig
	client        cherami.Client
	clientFactory internal.ClientFactory
	tracer        opentracing.Tracer
}

var _ transport.Transport = (*Transport)(nil)

// Start starts the cherami transport.
func (t *Transport) Start() error {
	return t.once.Start(func() error {
		if len(t.config.ServiceName) == 0 {
			return fmt.Errorf(`service name cannot be empty`)
		}

		serviceName := fmt.Sprintf("yarpc-cherami-%s", t.config.ServiceName)

		var err error
		if len(t.config.HyperbahnHostFile) > 0 {
			t.client, err = t.clientFactory.GetClientWithHyperbahn(serviceName, t.config.HyperbahnHostFile)

		} else {
			t.client, err = t.clientFactory.GetClientWithFrontEnd(serviceName, t.config.FrontendIP, t.config.Port)
		}
		return err
	})
}

// Stop stops the cherami transport.
func (t *Transport) Stop() error {
	return t.once.Stop(func() error {
		t.client.Close()
		return nil
	})
}

// IsRunning returns whether the cherami transport is running.
func (t *Transport) IsRunning() bool {
	return t.once.IsRunning()
}
