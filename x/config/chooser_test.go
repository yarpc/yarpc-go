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

package config

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/x/peerheap"
	"go.uber.org/yarpc/peer/x/roundrobin"
)

func TestChooserConfigurator(t *testing.T) {
	tests := []struct {
		desc    string
		given   string
		wantErr []string
		test    func(*testing.T, yarpc.Config)
	}{
		{
			desc: "all is well that ends well",
			given: expand(`
				transports:
					faketransport:
						address: ":1234"
				outbounds:
					theirservice:
						unary:
							faketransport:
								choose: fakechooser
								with: fakebinder
								watch: true
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound, ok := c.Outbounds["theirservice"]
				require.True(t, ok, "config has outbound")

				require.NotNil(t, outbound.Unary, "must have unary outbound")
				unary, ok := outbound.Unary.(*fakeOutbound)
				require.True(t, ok, "unary outbound must be fake outbound")

				transports := unary.Transports()
				require.Equal(t, len(transports), 1, "must have one transport")

				transport, ok := transports[0].(*fakeTransport)
				require.True(t, ok, "must be a fake transport")
				assert.Equal(t, transport.address, ":1234", "transport configured")

				require.NotNil(t, unary.chooser, "must have chooser")
				chooser, ok := unary.chooser.(*peer.BoundChooser)
				require.True(t, ok, "unary chooser must be a bound chooser")

				updater, ok := chooser.Updater().(*fakePeerListUpdater)
				require.True(t, ok, "updater is a faker binder")
				assert.True(t, updater.watch, "binder configured to watch")

				list, ok := chooser.ChooserList().(*fakeChooser)
				require.True(t, ok, "list is a fake chooser")
				_ = list

				dispatcher := yarpc.NewDispatcher(c)
				assert.NoError(t, dispatcher.Start(), "error starting")
				assert.NoError(t, dispatcher.Stop(), "error stopping")
			},
		},
		{
			desc: "use static peers with round robin and exercise choose",
			given: expand(`
				outbounds:
					theirservice:
						unary:
							faketransport:
								choose: round-robin
								peers:
								- 127.0.0.1:8080
								- 127.0.0.1:8081
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound := c.Outbounds["theirservice"]
				unary := outbound.Unary.(*fakeOutbound)
				transport := unary.Transports()[0].(*fakeTransport)
				chooser := unary.chooser.(*peer.BoundChooser)
				binder := chooser.Updater()
				list, ok := chooser.ChooserList().(*roundrobin.List)
				require.True(t, ok, "chooser least pending")
				_ = list

				// Attempt to choose a peer
				dispatcher := yarpc.NewDispatcher(c)
				require.NoError(t, dispatcher.Start(), "error starting dispatcher")
				defer func() {
					require.NoError(t, dispatcher.Stop(), "error stopping dispatcher")
				}()

				// TODO implement Lifecycle on Dispatcher
				//require.True(t, dispatcher.IsRunning(), "dispatcher is running")
				require.True(t, transport.IsRunning(), "transport is running")
				require.True(t, unary.IsRunning(), "outbound is running")
				require.True(t, list.IsRunning(), "chooser is running")
				require.True(t, binder.IsRunning(), "binder is running")

				ctx := context.Background()
				ctx, cancel := context.WithTimeout(ctx, time.Second)
				defer cancel()
				peer, onFinish, err := chooser.Choose(ctx, nil)
				require.NoError(t, err, "error choosing peer")
				defer onFinish(nil)

				assert.Equal(t, peer.Identifier(), "127.0.0.1:8080", "chooses first peer")
			},
		},
		{
			desc: "use default chooser",
			given: expand(`
				outbounds:
					theirservice:
						unary:
							faketransport:
								with: fakebinder
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound := c.Outbounds["theirservice"]
				unary := outbound.Unary.(*fakeOutbound)
				chooser := unary.chooser.(*peer.BoundChooser)
				list, ok := chooser.ChooserList().(*peerheap.List)
				require.True(t, ok, "chooser list peer heap by default")
				_ = list
			},
		},
		{
			desc: "use least-pending chooser",
			given: expand(`
				outbounds:
					theirservice:
						unary:
							faketransport:
								choose: least-pending
								with: fakebinder
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound := c.Outbounds["theirservice"]
				unary := outbound.Unary.(*fakeOutbound)
				chooser := unary.chooser.(*peer.BoundChooser)
				list, ok := chooser.ChooserList().(*peerheap.List)
				require.True(t, ok, "chooser list peer heap")
				_ = list
			},
		},
		{
			desc: "use round-robin chooser",
			given: expand(`
				outbounds:
					theirservice:
						unary:
							faketransport:
								choose: round-robin
								with: fakebinder
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound := c.Outbounds["theirservice"]
				unary := outbound.Unary.(*fakeOutbound)
				chooser := unary.chooser.(*peer.BoundChooser)
				list, ok := chooser.ChooserList().(*roundrobin.List)
				require.True(t, ok, "chooser round robin")
				_ = list
			},
		},
		{
			desc: "invalid choose with binder property",
			given: expand(`
				outbounds:
					theirservice:
						unary:
							faketransport:
								with: 10
			`),
			wantErr: []string{
				// TODO `failed to configure unary outbound for service "theirservice" over transport "faketransport": `,
				`failed to add outbound "theirservice": `,
				`failed to decode unary outbound configuration: `,
				`failed to decode *config.fakeOutboundConfig: `,
				// TODO dubious `error decoding '': `,
				`could not decode config.fakeOutboundConfig from config.attributeMap: `,
				`could not decode outbound peer list updater config, "with": `,
				`failed to read attribute "with": 10`,
			},
		},
		{
			desc: "invalid choose with binder property",
			given: expand(`
				outbounds:
					theirservice:
						unary:
							faketransport:
							choose: 10
			`),
			wantErr: []string{
				// TODO `failed to configure unary outbound for service "theirservice" over transport "faketransport": `,
				// TODO odd quoting
				`error decoding '[theirservice]': `,
				`could not decode config.outbounds from `,
				`failed to read unary outbound configuration: `,
				`failed to read attribute "unary": `,
				// TODO maybe should have line with "choose"
			},
		},
		{
			desc: "cannot combine peer and peers",
			given: expand(`
				outbounds:
					theirservice:
						unary:
							faketransport:
								peer: a
								peers:
								- a
								- b
			`),
			wantErr: []string{
				`failed to configure unary outbound for service "theirservice" over transport "faketransport": `,
				`cannot combine "peer": "a" and "peers": ["a" "b"]`,
			},
		},
		{
			desc: "extraneous config in combination with single peer",
			given: expand(`
				outbounds:
					theirservice:
						unary:
							faketransport:
								peer: a
								conspicuously: present
			`),
			wantErr: []string{
				`failed to configure unary outbound for service "theirservice" over transport "faketransport": `,
				`unrecognized attributes for outbound peer list/chooser config: `,
				`conspicuously`,
				`present`,
			},
		},
		{
			desc: "extraneous config in combination with peers",
			given: expand(`
				outbounds:
					theirservice:
						unary:
							faketransport:
								peers:
								- a
								conspicuously: present
			`),
			wantErr: []string{
				`failed to configure unary outbound for service "theirservice" over transport "faketransport": `,
				`unrecognized attributes for outbound peer list/chooser config: `,
				`conspicuously`,
				`present`,
			},
		},
		{
			desc: "extraneous config in combination with custom binder",
			given: expand(`
				outbounds:
					theirservice:
						unary:
							faketransport:
								with: fakebinder
								conspicuously: present
			`),
			wantErr: []string{
				`failed to configure unary outbound for service "theirservice" over transport "faketransport": `,
				`unrecognized attributes for outbound peer list/chooser config: `,
				`has invalid keys:`,
				`conspicuously`,
				// TODO would be nice to show `present`,
				// Map decoder error has empty quotes '' instead of representation of actual config map
			},
		},
		{
			desc: "missing peer binder config",
			given: expand(`
				outbounds:
					theirservice:
						unary:
							faketransport: {}
			`),
			wantErr: []string{
				`failed to configure unary outbound for service "theirservice" over transport "faketransport": `,
				`missing "peer", "peers", or "with" peer list binder`,
			},
		},
		{
			desc: "unsupported peer list binder",
			given: expand(`
				outbounds:
					theirservice:
						unary:
							faketransport:
								with: mind-control
			`),
			wantErr: []string{
				`failed to configure unary outbound for service "theirservice" over transport "faketransport": `,
				`not a supported peer list binder "mind-control"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			configer := fakeConfigurator()
			config, err := configer.LoadConfigFromYAML("fakeservice", strings.NewReader(tt.given))
			if len(tt.wantErr) > 0 {
				for _, wantErr := range tt.wantErr {
					require.Contains(t, err.Error(), wantErr, "expected error")
				}
			} else {
				require.NoError(t, err, "error loading config")
			}
			if tt.test != nil {
				tt.test(t, config)
			}
		})
	}
}
