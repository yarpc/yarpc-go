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

package http

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/yarpcconfig"
)

func TestTransportSpec(t *testing.T) {
	// This test is a cross-product of the transport, inbound and outbound
	// test assertions.
	//
	// Configuration, environment variables, and TransportSpec options are all
	// combined. If any entry had a non-empty wantErrors, any test case with
	// that entry is expected to fail.
	//
	// If the inbound and outbound tests state that they are both empty, the
	// test case will be skipped because we don't build a transport if there
	// is no inbound or outbound.

	type attrs map[string]interface{}

	type transportTest struct {
		desc string            // description
		cfg  attrs             // transport.http section of the config
		env  map[string]string // environment variables
		opts []Option          // transport spec options

		wantErrors []string
		wantClient *wantHTTPClient
	}

	type wantInbound struct {
		Address         string
		Mux             *http.ServeMux
		MuxPattern      string
		GrabHeaders     map[string]struct{}
		ShutdownTimeout time.Duration
	}

	type inboundTest struct {
		desc string            // description
		cfg  attrs             // inbounds.http section of the config
		env  map[string]string // environment variables
		opts []Option          // transport spec options

		empty bool // whether this test case is empty

		wantErrors  []string
		wantInbound *wantInbound
	}

	type wantOutbound struct {
		URLTemplate string
		Headers     http.Header
	}

	type outboundTest struct {
		desc string            // description
		cfg  attrs             // outbounds section of the config
		env  map[string]string // environment variables
		opts []Option          // transport spec options

		empty bool // whether this test case is empty

		wantErrors    []string
		wantOutbounds map[string]wantOutbound
	}

	transportTests := []transportTest{
		{
			desc: "no transport config",
			wantClient: &wantHTTPClient{
				KeepAlive:           30 * time.Second,
				MaxIdleConnsPerHost: 2,
				ConnTimeout:         defaultConnTimeout,
			},
		},
		{
			desc: "transport options",
			opts: []Option{
				KeepAlive(5 * time.Second),
				MaxIdleConnsPerHost(42),
			},
			wantClient: &wantHTTPClient{
				KeepAlive:           5 * time.Second,
				MaxIdleConnsPerHost: 42,
				ConnTimeout:         defaultConnTimeout,
			},
		},
		{
			desc: "explicit transport config",
			cfg: attrs{
				"keepAlive":             "5s",
				"maxIdleConns":          1,
				"maxIdleConnsPerHost":   2,
				"idleConnTimeout":       "5s",
				"connTimeout":           "1s",
				"disableKeepAlives":     true,
				"disableCompression":    true,
				"responseHeaderTimeout": "1s",
			},
			wantClient: &wantHTTPClient{
				KeepAlive:             5 * time.Second,
				MaxIdleConns:          1,
				MaxIdleConnsPerHost:   2,
				IdleConnTimeout:       5 * time.Second,
				ConnTimeout:           1 * time.Second,
				DisableKeepAlives:     true,
				DisableCompression:    true,
				ResponseHeaderTimeout: 1 * time.Second,
			},
		},
	}

	serveMux := http.NewServeMux()

	inboundTests := []inboundTest{
		{desc: "no inbound", empty: true},
		{
			desc:       "inbound without address",
			cfg:        attrs{},
			wantErrors: []string{"inbound address is required"},
		},
		{
			desc:        "simple inbound",
			cfg:         attrs{"address": ":8080"},
			wantInbound: &wantInbound{Address: ":8080", ShutdownTimeout: defaultShutdownTimeout},
		},
		{
			desc: "simple inbound with grab headers",
			cfg:  attrs{"address": ":8080", "grabHeaders": []string{"x-foo", "x-bar"}},
			wantInbound: &wantInbound{
				Address:         ":8080",
				GrabHeaders:     map[string]struct{}{"x-foo": {}, "x-bar": {}},
				ShutdownTimeout: defaultShutdownTimeout,
			},
		},
		{
			desc:        "inbound interpolation",
			cfg:         attrs{"address": "${HOST:}:${PORT}"},
			env:         map[string]string{"HOST": "127.0.0.1", "PORT": "80"},
			wantInbound: &wantInbound{Address: "127.0.0.1:80", ShutdownTimeout: defaultShutdownTimeout},
		},
		{
			desc: "serve mux",
			cfg:  attrs{"address": ":8080"},
			opts: []Option{
				Mux("/yarpc", serveMux),
			},
			wantInbound: &wantInbound{
				Address:         ":8080",
				Mux:             serveMux,
				MuxPattern:      "/yarpc",
				ShutdownTimeout: defaultShutdownTimeout,
			},
		},
		{
			desc:        "shutdown timeout",
			cfg:         attrs{"address": ":8080", "shutdownTimeout": "1s"},
			wantInbound: &wantInbound{Address: ":8080", ShutdownTimeout: time.Second},
		},
		{
			desc:        "shutdown timeout 0",
			cfg:         attrs{"address": ":8080", "shutdownTimeout": "0s"},
			wantInbound: &wantInbound{Address: ":8080", ShutdownTimeout: 0},
		},
		{
			desc:       "shutdown timeout err",
			cfg:        attrs{"address": ":8080", "shutdownTimeout": "-1s"},
			wantErrors: []string{`shutdownTimeout must not be negative, got: "-1s"`},
		},
	}

	outboundTests := []outboundTest{
		{desc: "no outbound", empty: true},
		{
			desc: "simple outbound",
			cfg: attrs{
				"myservice": attrs{
					"http": attrs{"url": "http://localhost:4040/yarpc"},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					URLTemplate: "http://localhost:4040/yarpc",
				},
			},
		},
		{
			desc: "outbound interpolation",
			env:  map[string]string{"ADDR": "127.0.0.1:80"},
			cfg: attrs{
				"myservice": attrs{
					"http": attrs{"url": "http://${ADDR}/yarpc"},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					URLTemplate: "http://127.0.0.1:80/yarpc",
				},
			},
		},
		{
			desc: "outbound url template option",
			opts: []Option{
				URLTemplate("http://127.0.0.1:8080/yarpc"),
			},
			cfg: attrs{
				"myservice": attrs{
					"http": attrs{"peer": "127.0.0.1:8888"},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					URLTemplate: "http://127.0.0.1:8080/yarpc",
				},
			},
		},
		{
			desc: "outbound url template option override",
			opts: []Option{
				URLTemplate("http://127.0.0.1:8080/yarpc"),
			},
			cfg: attrs{
				"myservice": attrs{
					"http": attrs{
						"url":  "http://host/yarpc/v1",
						"peer": "127.0.0.1:8888",
					},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					URLTemplate: "http://host/yarpc/v1",
				},
			},
		},
		{
			desc: "outbound header options",
			opts: []Option{
				AddHeader("X-Token", "token-1"),
				AddHeader("X-Token-2", "token-2"),
				AddHeader("X-Token", "token-3"),
			},
			cfg: attrs{
				"myservice": attrs{
					"http": attrs{"url": "http://localhost/"},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					URLTemplate: "http://localhost/",
					Headers: http.Header{
						"X-Token":   {"token-1", "token-3"},
						"X-Token-2": {"token-2"},
					},
				},
			},
		},
		{
			desc: "outbound header config",
			opts: []Option{
				AddHeader("X-Token", "token-1"),
			},
			cfg: attrs{
				"myservice": attrs{
					"http": attrs{
						"url": "http://localhost/",
						"addHeaders": attrs{
							"x-token":   "token-3",
							"X-Token-2": "token-2",
						},
					},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					URLTemplate: "http://localhost/",
					Headers: http.Header{
						"X-Token":   {"token-1", "token-3"},
						"X-Token-2": {"token-2"},
					},
				},
			},
		},
		{
			desc: "outbound header config with peer",
			cfg: attrs{
				"myservice": attrs{
					"http": attrs{
						"url":        "http://localhost/yarpc",
						"peer":       "127.0.0.1:8080",
						"addHeaders": attrs{"x-token": "token"},
					},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {
					URLTemplate: "http://localhost/yarpc",
					Headers: http.Header{
						"X-Token": {"token"},
					},
				},
			},
		},
		{
			desc: "outbound peer build error",
			cfg: attrs{
				"myservice": attrs{
					"http": attrs{
						"least-pending": []string{
							"127.0.0.1:8080",
							"127.0.0.1:8081",
							"127.0.0.1:8082",
						},
					},
				},
			},
			wantErrors: []string{
				"cannot configure peer chooser for HTTP outbound",
				`failed to read attribute "least-pending"`,
			},
		},
		{
			desc: "unknown preset",
			cfg: attrs{
				"myservice": attrs{
					"http": attrs{"with": "derp"},
				},
			},
			wantErrors: []string{
				`failed to configure unary outbound for "myservice":`,
				"cannot configure peer chooser for HTTP outbound:",
				`no recognized peer chooser preset "derp"`,
			},
		},
	}

	runTest := func(t *testing.T, trans transportTest, inbound inboundTest, outbound outboundTest) {
		env := make(map[string]string)
		for k, v := range trans.env {
			env[k] = v
		}
		for k, v := range inbound.env {
			_, ok := env[k]
			require.False(t, ok,
				"invalid test: environment variable %q is defined multiple times", k)
			env[k] = v
		}
		for k, v := range outbound.env {
			_, ok := env[k]
			require.False(t, ok,
				"invalid test: environment variable %q is defined multiple times", k)
			env[k] = v
		}
		configurator := yarpcconfig.New(yarpcconfig.InterpolationResolver(mapResolver(env)))

		opts := append(append(trans.opts, inbound.opts...), outbound.opts...)
		if trans.wantClient != nil {
			opts = append(opts, useFakeBuildClient(t, trans.wantClient))
		}
		err := configurator.RegisterTransport(TransportSpec(opts...))
		require.NoError(t, err, "failed to register transport spec")

		cfgData := make(attrs)
		if trans.cfg != nil {
			cfgData["transports"] = attrs{"http": trans.cfg}
		}
		if inbound.cfg != nil {
			cfgData["inbounds"] = attrs{"http": inbound.cfg}
		}
		if outbound.cfg != nil {
			cfgData["outbounds"] = outbound.cfg
		}
		cfg, err := configurator.LoadConfig("foo", cfgData)

		wantErrors := append(append(trans.wantErrors, inbound.wantErrors...), outbound.wantErrors...)
		if len(wantErrors) > 0 {
			require.Error(t, err, "expected failure while loading config %+v", cfgData)
			for _, msg := range wantErrors {
				assert.Contains(t, err.Error(), msg)
			}
			return
		}

		require.NoError(t, err, "expected success while loading config %+v", cfgData)

		if want := inbound.wantInbound; want != nil {
			assert.Len(t, cfg.Inbounds, 1, "expected exactly one inbound in %+v", cfgData)
			ib, ok := cfg.Inbounds[0].(*Inbound)
			if assert.True(t, ok, "expected *Inbound, got %T", cfg.Inbounds[0]) {
				assert.Equal(t, want.Address, ib.addr, "inbound address should match")
				assert.Equal(t, want.MuxPattern, ib.muxPattern,
					"inbound mux pattern should match")
				// == because we want it to be the same object
				assert.True(t, want.Mux == ib.mux, "inbound mux should match")
				// this has to be done because assert.Equal returns false if one map
				// is nil and the other is empty
				if len(want.GrabHeaders) > 0 {
					assert.Equal(t, want.GrabHeaders, ib.grabHeaders, "inbound grab headers should match")
				} else {
					assert.Empty(t, ib.grabHeaders)
				}
				assert.Equal(t, want.ShutdownTimeout, ib.shutdownTimeout, "shutdownTimeout should match")
			}
		}

		for svc, want := range outbound.wantOutbounds {
			ob, ok := cfg.Outbounds[svc].Unary.(*Outbound)
			if assert.True(t, ok, "expected *Outbound for %q, got %T", svc, cfg.Outbounds[svc].Unary) {
				// Verify that we install a oneway too
				_, ok := cfg.Outbounds[svc].Oneway.(*Outbound)
				assert.True(t, ok, "expected *Outbound for %q oneway, got %T", svc, cfg.Outbounds[svc].Oneway)

				assert.Equal(t, want.URLTemplate, ob.urlTemplate.String(), "outbound URLTemplate should match")
				assert.Equal(t, want.Headers, ob.headers, "outbound headers should match")
			}

		}
	}

	for _, transTT := range transportTests {
		for _, inboundTT := range inboundTests {
			for _, outboundTT := range outboundTests {
				// Special case: No inbounds or outbounds so we have nothing
				// to test.
				if inboundTT.empty && outboundTT.empty {
					continue
				}

				desc := fmt.Sprintf("%v/%v/%v", transTT.desc, inboundTT.desc, outboundTT.desc)
				t.Run(desc, func(t *testing.T) {
					runTest(t, transTT, inboundTT, outboundTT)
				})
			}
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

type wantHTTPClient struct {
	KeepAlive             time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	IdleConnTimeout       time.Duration
	DisableKeepAlives     bool
	DisableCompression    bool
	ResponseHeaderTimeout time.Duration
	ConnTimeout           time.Duration
}

// useFakeBuildClient verifies the configuration we use to build an HTTP
// client.
func useFakeBuildClient(t *testing.T, want *wantHTTPClient) TransportOption {
	return buildClient(func(options *transportOptions) *http.Client {
		assert.Equal(t, want.KeepAlive, options.keepAlive, "http.Client: KeepAlive should match")
		assert.Equal(t, want.MaxIdleConns, options.maxIdleConns, "http.Client: MaxIdleConns should match")
		assert.Equal(t, want.MaxIdleConnsPerHost, options.maxIdleConnsPerHost, "http.Client: MaxIdleConnsPerHost should match")
		// TODO(kris): not sure why the default is not zero.
		// assert.Equal(t, want.IdleConnTimeout, options.idleConnTimeout, "http.Client: IdleConnTimeout should match")
		assert.Equal(t, want.DisableKeepAlives, options.disableKeepAlives, "http.Client: DisableKeepAlives should match")
		assert.Equal(t, want.DisableCompression, options.disableCompression, "http.Client: DisableCompression should match")
		assert.Equal(t, want.ResponseHeaderTimeout, options.responseHeaderTimeout, "http.Client: ResponseHeaderTimeout should match")
		assert.Equal(t, want.ConnTimeout, options.connTimeout, "http.Client: ConnTimeout should match")
		return buildHTTPClient(options)
	})
}
