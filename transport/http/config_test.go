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

package http

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/yarpc/x/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransportSpec(t *testing.T) {
	type inbound struct {
		Address string
		Mux     *http.ServeMux
	}

	type outbound struct {
		URLTemplate string
	}

	type outbounds map[string]outbound

	serveMux := http.NewServeMux()

	tests := []struct {
		desc string
		env  map[string]string
		opts []Option
		yaml string

		wantErrors    []string
		wantInbound   *inbound
		wantOutbounds outbounds
	}{
		{
			desc:       "inbound without address",
			yaml:       "inbounds: {http: {}}",
			wantErrors: []string{"inbound address is required"},
		},
		{
			desc: "inbound",
			yaml: expand(`
				inbounds:
					http: {address: ":8080"}
			`),
			wantInbound: &inbound{Address: ":8080"},
		},
		{
			desc: "inbound interpolation",
			yaml: expand(`
				inbounds:
					http: {address: "${HOST:}:${PORT}"}
			`),
			env:         map[string]string{"HOST": "127.0.0.1", "PORT": "80"},
			wantInbound: &inbound{Address: "127.0.0.1:80"},
		},
		{
			desc: "inbound options",
			yaml: expand(`
				inbounds:
					http: {address: ":8080"}
			`),
			opts: []Option{Mux("/", serveMux)},
			wantInbound: &inbound{
				Address: ":8080",
				Mux:     serveMux,
			},
		},
		{
			desc: "outbound",
			yaml: expand(`
				outbounds:
					myservice:
						http:
							url: http://127.0.0.1:80/yarpc
			`),
			wantOutbounds: outbounds{
				"myservice": {
					URLTemplate: "http://127.0.0.1:80/yarpc",
				},
			},
		},
		{
			desc: "outbound interpolation",
			env:  map[string]string{"ADDR": "127.0.0.1:80"},
			yaml: expand(`
				outbounds:
					myservice:
						http:
							url: http://${ADDR}/yarpc
			`),
			wantOutbounds: outbounds{
				"myservice": {
					URLTemplate: "http://127.0.0.1:80/yarpc",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			configurator := config.New(config.InterpolationResolver(mapResolver(tt.env)))
			err := configurator.RegisterTransport(TransportSpec(tt.opts...))
			require.NoError(t, err, "failed to register transport spec")

			cfg, err := configurator.LoadConfigFromYAML("foo", bytes.NewBufferString(tt.yaml))
			if len(tt.wantErrors) > 0 {
				require.Error(t, err)
				for _, msg := range tt.wantErrors {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}

			require.NoError(t, err)

			if want := tt.wantInbound; want != nil {
				ib, ok := cfg.Inbounds[0].(*Inbound)
				require.True(t, ok, "expected *Inbound, got %T", cfg.Inbounds[0])

				assert.Equal(t, want.Address, ib.addr)
				assert.True(t, want.Mux == ib.mux, "ServeMux must be the same")
			}

			for svc, want := range tt.wantOutbounds {
				ob, ok := cfg.Outbounds[svc].Unary.(*Outbound)
				require.True(t, ok, "expected *Outbound, got %T", cfg.Outbounds[svc].Unary)

				assert.Equal(t, want.URLTemplate, ob.urlTemplate.String())
			}
		})
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

// TODO: Use whitespace.Expand after #956 merges
func expand(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = expandLine(l)
	}
	return strings.Join(lines, "\n")
}

func expandLine(l string) string {
	for i, c := range l {
		if c != '\t' {
			return strings.Repeat("  ", i) + l[i:]
		}
	}
	return ""
}
