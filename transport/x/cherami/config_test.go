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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/cherami-client-go/client/cherami"
	"go.uber.org/yarpc/x/config"
)

func TestParseIPAndPort(t *testing.T) {
	tests := []struct {
		give     string
		wantIP   string
		wantPort int
		wantErr  string
	}{
		{give: "", wantErr: "address is unspecified"},
		{
			give:    "hi",
			wantErr: `invalid address "hi": port was not specified`,
		},
		{
			give:    "localhost:",
			wantErr: `invalid port "" in address "localhost:":`,
		},
		{
			give:    "localhost:hi",
			wantErr: `invalid port "hi" in address "localhost:hi":`,
		},
		{
			give:     "localhost:4242",
			wantIP:   "localhost",
			wantPort: 4242,
		},
	}

	for _, tt := range tests {
		t.Run(tt.give, func(t *testing.T) {
			ip, port, err := parseIPAndPort(tt.give)

			if tt.wantErr != "" {
				require.Error(t, err, "expected failure")
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantIP, ip, "ip must match")
			assert.Equal(t, tt.wantPort, port, "port must match")
		})
	}
}

func TestTransportSpec(t *testing.T) {
	// This test is a cross-product of transport, inbound and outbound test
	// assertions.
	//
	//   transportTests x inboundTests x outboundTests
	//
	// Configuration, environment variables, and TransportSpec options are
	// combined. If any of the entries had non-empty wantErrors, any test case
	// with that entry is expected to fail.
	//
	// Further, the inbound and outbound tests may state that they are
	// "empty". If they're both empty, the test case will be skipped because a
	// transport is us

	type attrs map[string]interface{}

	type transportTest struct {
		desc string                // description
		cfg  attrs                 // transports.cherami section of the config
		env  map[string]string     // environment variables
		opts []TransportSpecOption // transport spec options

		wantErrors []string

		// If non-nil, we expect either a cherami.NewClient call or
		// cherami.NewHyperbahnClient call.
		wantClient          *fakeCheramiClient
		wantHyperbahnClient *fakeCheramiHyperbahnClient
	}

	type wantInbound struct {
		Destination   string
		ConsumerGroup string
		PrefetchCount int
	}

	type inboundTest struct {
		desc string                // description
		cfg  attrs                 // inbounds.cherami section of the config
		env  map[string]string     // environment variables
		opts []TransportSpecOption // transport spec options

		// Whether this test case is empty. If both, the inbound and the
		// outbound are empty, that test case will be skipped.
		empty bool

		wantErrors []string

		// If non-nil, we expect a Cherami inbound to be constructed with the
		// given properties.
		wantInbound *wantInbound
	}

	type wantOutbound struct {
		Destination string
	}

	type outboundTest struct {
		desc string                // description
		cfg  attrs                 // outbounds section of the config
		env  map[string]string     // environment variables
		opts []TransportSpecOption // transport spec options

		// Whether this test case is empty. If both, the inbound and the
		// outbound are empty, that test case will be skipped.
		empty bool

		wantErrors []string

		// If non-empty, we expect cherami outbounds to be constructed for the
		// given services with the specified configurations.
		wantOutbounds map[string]wantOutbound
	}

	transportTests := []transportTest{
		{
			desc:       "no transport",
			wantErrors: []string{`either an "address" or a "peerList" must be specified`},
		},
		{
			desc:       "invalid address",
			cfg:        attrs{"address": "hi"},
			wantErrors: []string{`invalid address "hi": port was not specified`},
		},
		{
			desc: "cherami hyperbahn client",
			opts: []TransportSpecOption{DefaultPeerList("hosts.json")},
			wantHyperbahnClient: &fakeCheramiHyperbahnClient{
				WantBootstrapFile: "hosts.json",
			},
		},
		{
			desc: "bad cherami hyperbahn client",
			opts: []TransportSpecOption{DefaultPeerList("hosts.json")},
			wantHyperbahnClient: &fakeCheramiHyperbahnClient{
				WantBootstrapFile: "hosts.json",
				RaiseError:        errors.New("great sadness"),
			},
			wantErrors: []string{
				`failed to create Cherami client with peer list "hosts.json": great sadness`,
			},
		},
		{
			desc: "cherami hyperbahn client explicit peer list",
			cfg:  attrs{"peerList": "hosts.dev.json"},
			opts: []TransportSpecOption{DefaultPeerList("hosts.json")},
			wantHyperbahnClient: &fakeCheramiHyperbahnClient{
				WantBootstrapFile: "hosts.dev.json",
			},
		},
		{
			desc: "cherami local client",
			cfg:  attrs{"address": "127.0.0.1:4922"},
			wantClient: &fakeCheramiClient{
				WantHost: "127.0.0.1",
				WantPort: 4922,
			},
		},
		{
			desc: "bad cherami local client",
			cfg:  attrs{"address": "127.0.0.1:4922"},
			wantClient: &fakeCheramiClient{
				WantHost:   "127.0.0.1",
				WantPort:   4922,
				RaiseError: errors.New("great sadness"),
			},
			wantErrors: []string{
				`failed to create Cherami client with address "127.0.0.1:4922": great sadness`,
			},
		},

		// The following group of tests use the same configuration but
		// different environment variables.
		{
			desc: "transport interpolation: hyperbahn",
			opts: []TransportSpecOption{DefaultPeerList("hosts.json")},
			cfg: attrs{
				"address":       "${CHERAMI_ADDRESS:}",
				"peerList":      "${CHERAMI_PEER_LIST:}",
				"deploymentStr": "${CHERAMI_DEPLOYMENT:prod}",
			},
			wantHyperbahnClient: &fakeCheramiHyperbahnClient{
				WantBootstrapFile: "hosts.json",
				WantOptions: cherami.ClientOptions{
					DeploymentStr: "prod",
				},
			},
		},
		{
			desc: "transport interpolation: local",
			opts: []TransportSpecOption{DefaultPeerList("hosts.json")},
			env: map[string]string{
				"CHERAMI_ADDRESS":    "myserver.local:4922",
				"CHERAMI_DEPLOYMENT": "dev",
			},
			cfg: attrs{
				"address":       "${CHERAMI_ADDRESS:}",
				"peerList":      "${CHERAMI_PEER_LIST:}",
				"deploymentStr": "${CHERAMI_DEPLOYMENT:prod}",
			},
			wantClient: &fakeCheramiClient{
				WantHost: "myserver.local",
				WantPort: 4922,
				WantOptions: cherami.ClientOptions{
					DeploymentStr: "dev",
				},
			},
		},
	}

	inboundTests := []inboundTest{
		{desc: "no inbound", empty: true},
		{
			desc: "default inbound",
			cfg:  attrs{},
			wantInbound: &wantInbound{
				Destination:   "/foo/yarpc_dest",
				ConsumerGroup: "/foo/yarpc_cg",
				PrefetchCount: 10,
			},
		},
		{
			desc: "explicit inbound",
			cfg: attrs{
				"destination":   "/bar/dest",
				"consumerGroup": "/bar/cg",
			},
			wantInbound: &wantInbound{
				Destination:   "/bar/dest",
				ConsumerGroup: "/bar/cg",
				PrefetchCount: 10,
			},
		},
		{
			desc: "inbound interpolation",
			cfg: attrs{
				"destination":   "/baz/yarpc-dest-${NAME}",
				"consumerGroup": "/baz/yarpc-cg-${NAME}",
				"prefetchCount": "42",
			},
			env: map[string]string{"NAME": "hello"},
			wantInbound: &wantInbound{
				Destination:   "/baz/yarpc-dest-hello",
				ConsumerGroup: "/baz/yarpc-cg-hello",
				PrefetchCount: 42,
			},
		},
	}

	outboundTests := []outboundTest{
		{desc: "no outbound", empty: true},
		{
			desc: "default outbound",
			cfg:  attrs{"myservice": attrs{"cherami": attrs{}}},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {Destination: "/foo/yarpc_dest"},
			},
		},
		{
			desc: "explicit outbound",
			cfg: attrs{
				"myservice": attrs{
					"cherami": attrs{"destination": "/bar/dest"},
				},
			},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {Destination: "/bar/dest"},
			},
		},
		{
			desc: "outbound interpolation",
			cfg: attrs{
				"myservice": attrs{
					"cherami": attrs{"destination": "/baz/yarpc-dest-${MYSERVICE_NAME}"},
				},
			},
			env: map[string]string{"MYSERVICE_NAME": "hi"},
			wantOutbounds: map[string]wantOutbound{
				"myservice": {Destination: "/baz/yarpc-dest-hi"},
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
		configurator := config.New(config.InterpolationResolver(mapResolver(env)))

		opts := append(append(trans.opts, inbound.opts...), outbound.opts...)
		if trans.wantClient != nil {
			opts = append(opts, useFakeCheramiClient(t, trans.wantClient))
		}
		if trans.wantHyperbahnClient != nil {
			opts = append(opts, useFakeCheramiHyperbahnClient(t, trans.wantHyperbahnClient))
		}
		err := configurator.RegisterTransport(TransportSpec(opts...))
		require.NoError(t, err, "failed to register transport spec")

		cfgData := make(attrs)
		if trans.cfg != nil {
			cfgData["transports"] = attrs{"cherami": trans.cfg}
		}
		if inbound.cfg != nil {
			cfgData["inbounds"] = attrs{"cherami": inbound.cfg}
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
				assert.Equal(t, want.Destination, ib.opts.Destination,
					"inbound destination should match")
				assert.Equal(t, want.ConsumerGroup, ib.opts.ConsumerGroup,
					"inbound consumer group should match")
				assert.Equal(t, want.PrefetchCount, ib.opts.PrefetchCount,
					"inbound prefetch count should match")
			}
		}

		for svc, want := range outbound.wantOutbounds {
			ob, ok := cfg.Outbounds[svc].Oneway.(*Outbound)
			if assert.True(t, ok, "expected *Outbound for %q, got %T", svc, cfg.Outbounds[svc].Oneway) {
				assert.Equal(t, want.Destination, ob.opts.Destination,
					"outbound destination for %q should match", svc)
			}
		}
	}

	for _, transTT := range transportTests {
		for _, inboundTT := range inboundTests {
			for _, outboundTT := range outboundTests {
				// Special case: No inbounds and outbounds so we have nothing
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

type fakeCheramiClient struct {
	WantHost    string
	WantPort    int
	WantOptions cherami.ClientOptions
	RaiseError  error
}

// Build a cheramiNewClientFunc which expects the given arguments and
// delegates to the real cherami.NewClient function.
func useFakeCheramiClient(t *testing.T, c *fakeCheramiClient) TransportSpecOption {
	return cheramiNewClient(
		func(serviceName string, host string, port int, options *cherami.ClientOptions) (cherami.Client, error) {
			assert.Equal(t, c.WantHost, host, "cherami.NewClient: host must match")
			assert.Equal(t, c.WantPort, port, "cherami.NewClient: port must match")
			assert.Equal(t, &c.WantOptions, options, "cherami.NewClient: options must match")

			if c.RaiseError != nil {
				return nil, c.RaiseError
			}

			return cherami.NewClient(serviceName, host, port, options)
		})
}

type fakeCheramiHyperbahnClient struct {
	WantBootstrapFile string
	WantOptions       cherami.ClientOptions
	RaiseError        error
}

// Build a cheramiNewHyperbahnClientFunc which expects the given arguments and
// delegates to the real cherami.NewHyperbahnClient function.
func useFakeCheramiHyperbahnClient(t *testing.T, c *fakeCheramiHyperbahnClient) TransportSpecOption {
	return cheramiNewHyperbahnClient(
		func(serviceName string, bootstrapFile string, options *cherami.ClientOptions) (cherami.Client, error) {
			assert.Equal(t, c.WantBootstrapFile, bootstrapFile, "cherami.NewHyperbahnClient: bootstrapFile must match")
			assert.Equal(t, &c.WantOptions, options, "cherami.NewHyperbahnClient: options must match")

			if c.RaiseError != nil {
				return nil, c.RaiseError
			}

			return cherami.NewHyperbahnClient(serviceName, bootstrapFile, options)
		})
}

func mapResolver(m map[string]string) func(string) (string, bool) {
	return func(k string) (v string, ok bool) {
		if m != nil {
			v, ok = m[k]
		}
		return
	}
}
