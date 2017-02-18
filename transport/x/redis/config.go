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

package redis

import (
	"errors"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/x/config"
)

// TransportConfig configures the shared Redis transport. This is shared
// between all Redis outbounds and inbounds of a Dispatcher.
type TransportConfig struct {
	Address string `config:"address"`
}

// InboundConfig configures a Redis oneway inbound.
type InboundConfig struct {
	QueueKey      string        `config:"queueKey"`
	ProcessingKey string        `config:"processingKey"`
	Timeout       time.Duration `config:"timeout"`
}

// OutboundConfig configures a Redis oneway outbound.
type OutboundConfig struct {
	QueueKey string `config:"queueKey"`
}

// TransportSpec returns a TransportSpec for the Redis oneway transport. See
// TransportConfig, InboundConfig, and OutboundConfig for details on the
// various supported configuration parameters.
func TransportSpec() config.TransportSpec {
	return config.TransportSpec{
		Name:                "redis",
		BuildTransport:      buildTransport,
		BuildInbound:        buildInbound,
		BuildOnewayOutbound: buildOnewayOutbound,
	}
}

func buildTransport(tc *TransportConfig) (transport.Transport, error) {
	if tc.Address == "" {
		return nil, errors.New("address is required")
	}
	return NewRedis5Client(tc.Address), nil
}

func buildOnewayOutbound(oc *OutboundConfig, t transport.Transport) (transport.OnewayOutbound, error) {
	if oc.QueueKey == "" {
		return nil, errors.New("queue key is required")
	}

	return NewOnewayOutbound(t.(Client), oc.QueueKey), nil
}

func buildInbound(ic *InboundConfig, t transport.Transport) (transport.Inbound, error) {
	if ic.QueueKey == "" {
		return nil, errors.New("queue key is required")
	}

	if ic.ProcessingKey == "" {
		return nil, errors.New("processing key is required")
	}

	if ic.Timeout == 0 {
		ic.Timeout = time.Second
	}

	return NewInbound(t.(Client), ic.QueueKey, ic.ProcessingKey, ic.Timeout), nil
}

// TODO: Document configuratior parameters
