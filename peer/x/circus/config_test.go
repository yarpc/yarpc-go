// Copyright (c) 2020 Uber Technologies, Inc.
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

package circus

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/whitespace"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/yarpc/yarpctest"
)

type attrs map[string]interface{}

func TestConfig(t *testing.T) {
	cfg := yarpcconfig.New()
	cfg.RegisterPeerList(Spec())
	cfg.RegisterTransport(yarpctest.FakeTransportSpec())
	config, err := cfg.LoadConfig("our-service", attrs{
		"outbounds": attrs{
			"their-service": attrs{
				"fake-transport": attrs{
					"circus": attrs{
						"failFast": true,
						"peers": []string{
							"1.1.1.1:1111",
							"2.2.2.2:2222",
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, config.Outbounds)
	require.NotNil(t, config.Outbounds["their-service"])
	require.NotNil(t, config.Outbounds["their-service"].Unary)
}

func TestFailFastConfig(t *testing.T) {
	conn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	serviceName := "test"
	config := whitespace.Expand(fmt.Sprintf(`
		outbounds:
			nowhere:
				http:
					circus:
						peers:
							- %q
						failFast: true
	`, conn.Addr()))
	cfgr := yarpcconfig.New()
	cfgr.MustRegisterTransport(http.TransportSpec())
	cfgr.MustRegisterPeerList(Spec())
	cfg, err := cfgr.LoadConfigFromYAML(serviceName, strings.NewReader(config))
	require.NoError(t, err)

	d := yarpc.NewDispatcher(cfg)
	d.Start()
	defer d.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	client := d.MustOutboundConfig("nowhere")
	_, err = client.Outbounds.Unary.Call(ctx, &transport.Request{
		Service:   "service",
		Caller:    "caller",
		Encoding:  transport.Encoding("blank"),
		Procedure: "bogus",
		Body:      strings.NewReader("nada"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no peer available")
}
