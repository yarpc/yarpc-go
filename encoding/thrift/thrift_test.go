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

package thrift_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/examples/thrift-hello/hello/echo"
	"go.uber.org/yarpc/internal/examples/thrift-hello/hello/echo/helloclient"
	"go.uber.org/yarpc/internal/examples/thrift-hello/hello/echo/helloserver"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/transport/tchannel"
	. "go.uber.org/yarpc/x/yarpctest"
	"go.uber.org/yarpc/x/yarpctest/api"
	"go.uber.org/yarpc/x/yarpctest/types"
)

func TestThrift(t *testing.T) {
	p := NewPortProvider(t)
	tests := []struct {
		name     string
		services Lifecycle
		requests Action
	}{
		{
			name: "stream requests",
			services: Lifecycles(
				TChannelService(
					Name("myservice"),
					p.NamedPort("1"),
					ThriftEchoProcedures(),
				),
			),
			requests: ConcurrentAction(
				RepeatAction(
					RandomizedTChannelEchoAction(
						"myservice",
						p.NamedPort("1"),
					),
					100,
				),
				10,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.services.Start(t))
			tt.requests.Run(t)
			require.NoError(t, tt.services.Stop(t))
		})
	}
}

func ThriftEchoProcedures() api.ServiceOption {
	return api.ServiceOptionFunc(func(opts *api.ServiceOpts) {
		opts.Procedures = append(opts.Procedures, helloserver.New(&helloHandler{})...)
	})
}

type helloHandler struct{}

func (h helloHandler) Echo(ctx context.Context, e *echo.EchoRequest) (*echo.EchoResponse, error) {
	return &echo.EchoResponse{Message: e.Message, Count: e.Count + 1}, nil
}

// RandomizedEchoAction creates a random echo action, and calls into the
// endpoint.
func RandomizedTChannelEchoAction(service string, p *types.Port) api.Action {
	return api.ActionFunc(func(t testing.TB) {
		cc, stop := getOutboundConfig(t, service, p)
		defer stop()

		client := helloclient.New(cc)
		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		defer cancel()
		count := Intn(500) + 1
		message := String(count)
		resp, err := client.Echo(ctx, &echo.EchoRequest{Message: message, Count: int16(count)})
		require.NoError(t, err)
		require.Equal(t, int16(count+1), resp.Count)
		require.Equal(t, message, resp.Message)
	})
}

func getOutboundConfig(t testing.TB, service string, port *types.Port) (c *transport.OutboundConfig, stop func()) {
	trans, err := tchannel.NewTransport(tchannel.ServiceName("caller"))
	require.NoError(t, err)
	out := trans.NewSingleOutbound(fmt.Sprintf(port.Listener.Addr().String()))

	require.NoError(t, trans.Start())
	require.NoError(t, out.Start())

	return &transport.OutboundConfig{
			CallerName: "caller",
			Outbounds: transport.Outbounds{
				ServiceName: service,
				Unary:       out,
			},
		}, func() {
			assert.NoError(t, trans.Stop())
			assert.NoError(t, out.Stop())
		}
}

// Randomization helpers
var randLock sync.Mutex
var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func Intn(length int) int {
	randLock.Lock()
	r := seededRand.Intn(length)
	randLock.Unlock()
	return r
}

func String(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[Intn(len(charset))]
	}
	return string(b)
}
