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

package tchannel

import (
	"fmt"
	"testing"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/x/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tchanneltest "github.com/uber/tchannel-go/testutils"
)

type badOption struct{}

func (badOption) tchannelOption() {}

func TestTransportSpecInvalidOption(t *testing.T) {
	assert.Panics(t, func() {
		TransportSpec(badOption{})
	})
}

func TestTransportSpec(t *testing.T) {
	someChannel := tchanneltest.NewServer(t, nil)
	defer someChannel.Close()

	type attrs map[string]interface{}

	type wantTransport struct {
		Address string
	}

	type inboundTest struct {
		desc string            // description
		cfg  attrs             // inbounds section of the config
		env  map[string]string // environment variables
		opts []Option          // transport spec options

		empty bool // whether this test case is empty

		wantErrors []string

		// Inbounds don't have any properties that affect the Inbound object.
		// We'll assert things about the transport only.
		wantTransport *wantTransport
	}

	type outboundTest struct {
		desc string            // description
		cfg  attrs             // outbounds section of the config
		env  map[string]string // environment variables
		opts []Option          // transport spec options

		empty bool // whether this test case is empty

		wantErrors    []string
		wantOutbounds []string
	}

	inboundTests := []inboundTest{
		{desc: "no inbound", empty: true},
		{
			desc:          "simple inbound",
			cfg:           attrs{"tchannel": attrs{"address": ":4040"}},
			wantTransport: &wantTransport{Address: ":4040"},
		},
		{
			desc:          "inbound interpolation",
			cfg:           attrs{"tchannel": attrs{"address": ":${PORT}"}},
			env:           map[string]string{"PORT": "4041"},
			wantTransport: &wantTransport{Address: ":4041"},
		},
		{
			desc:       "empty address",
			cfg:        attrs{"tchannel": attrs{"address": ""}},
			wantErrors: []string{"inbound address is required"},
		},
		{
			desc:       "missing address",
			cfg:        attrs{"tchannel": attrs{}},
			wantErrors: []string{"inbound address is required"},
		},
		{
			desc: "too many inbounds",
			cfg: attrs{
				"tchannel":  attrs{"address": ":4040"},
				"tchannel2": attrs{"address": ":4041", "type": "tchannel"},
			},
			wantErrors: []string{"at most one TChannel inbound may be specified"},
		},
		{
			desc:       "WithChannel fails",
			cfg:        attrs{"tchannel": attrs{"address": ":4040"}},
			opts:       []Option{WithChannel(someChannel)},
			wantErrors: []string{"TChannel TransportSpec does not accept WithChannel"},
		},
		{
			desc:       "ServiceName fails",
			cfg:        attrs{"tchannel": attrs{"address": ":4040"}},
			opts:       []Option{ServiceName("zzzzzzzzz")},
			wantErrors: []string{"TChannel TransportSpec does not accept ServiceName"},
		},
		{
			desc:       "ListenAddr fails",
			cfg:        attrs{"tchannel": attrs{"address": ":4040"}},
			opts:       []Option{ListenAddr(":8080")},
			wantErrors: []string{"TChannel TransportSpec does not accept ListenAddr"},
		},
	}

	outboundTests := []outboundTest{
		{desc: "no outbound", empty: true},
		{
			desc: "simple outbound",
			cfg: attrs{
				"myservice": attrs{
					"tchannel": attrs{
						"peer": "127.0.0.1:4040",
					},
				},
			},
		},
		{
			desc: "outbound interpolation",
			env:  map[string]string{"SERVICE_PORT": "4040"},
			cfg: attrs{
				"myservice": attrs{
					"tchannel": attrs{
						"peer": "127.0.0.1:${SERVICE_PORT}",
					},
				},
			},
		},
		{
			desc: "outbound bad peer list",
			cfg: attrs{
				"myservice": attrs{
					"tchannel": attrs{"least-pending": "wat"},
				},
			},
			wantErrors: []string{
				`failed to configure unary outbound for "myservice"`,
				`failed to read attribute "least-pending": wat`,
			},
		},
	}

	runTest := func(t *testing.T, inbound inboundTest, outbound outboundTest) {
		env := make(map[string]string)
		for k, v := range inbound.env {
			env[k] = v
		}
		for k, v := range outbound.env {
			_, ok := env[k]
			require.False(t, ok,
				"invalid test: environment variable %q is defined multiple times", k)
			env[k] = v
		}
		configurator := config.New(config.InterpolationResolver(mapResolver(env)))

		opts := append(inbound.opts, outbound.opts...)
		err := configurator.RegisterTransport(TransportSpec(opts...))
		require.NoError(t, err, "failed to register transport spec")

		cfgData := make(attrs)
		if inbound.cfg != nil {
			cfgData["inbounds"] = inbound.cfg
		}
		if outbound.cfg != nil {
			cfgData["outbounds"] = outbound.cfg
		}
		cfg, err := configurator.LoadConfig("foo", cfgData)

		if len(inbound.wantErrors) > 0 {
			require.Error(t, err, "expected failure while loading config %+v", cfgData)
			for _, msg := range inbound.wantErrors {
				assert.Contains(t, err.Error(), msg)
			}
			return
		}

		if len(outbound.wantErrors) > 0 {
			require.Error(t, err, "expected failure while loading config %+v", cfgData)
			for _, msg := range outbound.wantErrors {
				assert.Contains(t, err.Error(), msg)
			}
			return
		}

		require.NoError(t, err, "expected success while loading config %+v", cfgData)
		if want := inbound.wantTransport; want != nil {
			assert.Len(t, cfg.Inbounds, 1, "expected exactly one inbound in %+v", cfgData)
			ib, ok := cfg.Inbounds[0].(*Inbound)
			if assert.True(t, ok, "expected *Inbound, got %T", cfg.Inbounds[0]) {
				trans := ib.transport
				assert.Equal(t, "foo", trans.name, "service name must match")
				assert.Equal(t, want.Address, trans.addr, "transport address must match")
			}
		}

		for _, svc := range outbound.wantOutbounds {
			_, ok := cfg.Outbounds[svc].Unary.(*Outbound)
			assert.True(t, ok, "expected *Outbound for %q, got %T", svc, cfg.Outbounds[svc].Unary)
		}

		d := yarpc.NewDispatcher(cfg)
		require.NoError(t, d.Start(), "failed to start dispatcher")
		require.NoError(t, d.Stop(), "failed to stop dispatcher")
	}

	for _, inboundTT := range inboundTests {
		for _, outboundTT := range outboundTests {
			// Special case: No inbounds or outbounds so we have nothing to
			// test.
			if inboundTT.empty && outboundTT.empty {
				continue
			}

			desc := fmt.Sprintf("%v/%v", inboundTT.desc, outboundTT.desc)
			t.Run(desc, func(t *testing.T) {
				runTest(t, inboundTT, outboundTT)
			})
		}
	}
}

func mapResolver(m map[string]string) func(string) (string, bool) {
	return func(k string) (v string, ok bool) {
		if m != nil {
			v, ok = m[k]
		}
		return
	}
}
