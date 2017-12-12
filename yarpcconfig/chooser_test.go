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

package yarpcconfig_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	peerapi "go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/peer/peertest"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/interpolate"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/whitespace"
	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/peer/roundrobin"
	"go.uber.org/yarpc/peer/x/peerheap"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/yarpc/yarpctest"
)

func TestChooserConfigurator(t *testing.T) {
	tests := []struct {
		desc    string
		given   string
		env     map[string]string
		wantErr []string
		test    func(*testing.T, yarpc.Config)
	}{
		{
			desc: "single static peer",
			given: whitespace.Expand(`
				transports:
					fake-transport:
						nop: ":1234"
				outbounds:
					their-service:
						unary:
							fake-transport:
								nop: "*.*"
								peer: 127.0.0.1:8080
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound, ok := c.Outbounds["their-service"]
				require.True(t, ok, "config has outbound")

				require.NotNil(t, outbound.Unary, "must have unary outbound")
				unary, ok := outbound.Unary.(*yarpctest.FakeOutbound)
				require.True(t, ok, "unary outbound must be fake outbound")
				assert.Equal(t, "*.*", unary.NopOption(), "must have configured pattern")

				transports := unary.Transports()
				require.Equal(t, 1, len(transports), "must have one transport")

				transport, ok := transports[0].(*yarpctest.FakeTransport)
				require.True(t, ok, "must be a fake transport")
				assert.Equal(t, ":1234", transport.NopOption(), "transport configured")

				require.NotNil(t, unary.Chooser(), "must have chooser")
				chooser, ok := unary.Chooser().(*peer.Single)
				require.True(t, ok, "unary chooser must be a single peer chooser")

				dispatcher := yarpc.NewDispatcher(c)
				assert.NoError(t, dispatcher.Start(), "error starting")
				assert.NoError(t, dispatcher.Stop(), "error stopping")

				_ = chooser
			},
		},
		{
			desc: "custom chooser",
			given: whitespace.Expand(`
				transports:
					fake-transport:
						nop: ":1234"
				outbounds:
					their-service:
						unary:
							fake-transport:
								fake-chooser:
									nop: "*.*"
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound, ok := c.Outbounds["their-service"]
				require.True(t, ok, "config has outbound")

				require.NotNil(t, outbound.Unary, "must have unary outbound")
				unary, ok := outbound.Unary.(*yarpctest.FakeOutbound)
				require.True(t, ok, "unary outbound must be fake outbound")

				transports := unary.Transports()
				require.Equal(t, len(transports), 1, "must have one transport")

				transport, ok := transports[0].(*yarpctest.FakeTransport)
				require.True(t, ok, "must be a fake transport")
				assert.Equal(t, transport.NopOption(), ":1234", "transport configured")

				require.NotNil(t, unary.Chooser(), "must have chooser")
				chooser, ok := unary.Chooser().(*yarpctest.FakePeerChooser)
				require.True(t, ok, "unary chooser must be a fake peer chooser")
				require.Equal(t, "*.*", chooser.Nop())

				dispatcher := yarpc.NewDispatcher(c)
				assert.NoError(t, dispatcher.Start(), "error starting")
				assert.NoError(t, dispatcher.Stop(), "error stopping")

				_ = chooser
			},
		},
		{
			desc: "multiple static peers",
			given: whitespace.Expand(`
				transports:
					fake-transport:
						nop: ":1234"
				outbounds:
					their-service:
						unary:
							fake-transport:
								nop: "*.*"
								fake-list:
									peers:
										- 127.0.0.1:8080
										- 127.0.0.1:8081
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound, ok := c.Outbounds["their-service"]
				require.True(t, ok, "config has outbound")

				require.NotNil(t, outbound.Unary, "must have unary outbound")
				unary, ok := outbound.Unary.(*yarpctest.FakeOutbound)
				require.True(t, ok, "unary outbound must be fake outbound")

				transports := unary.Transports()
				require.Equal(t, len(transports), 1, "must have one transport")

				transport, ok := transports[0].(*yarpctest.FakeTransport)
				require.True(t, ok, "must be a fake transport")
				assert.Equal(t, transport.NopOption(), ":1234", "transport configured")

				require.NotNil(t, unary.Chooser(), "must have chooser")
				chooser, ok := unary.Chooser().(*peer.BoundChooser)
				require.True(t, ok, "unary chooser must be a bound chooser")

				updater, ok := chooser.Updater().(*peer.PeersUpdater)
				require.True(t, ok, "updater is a static peer list updater")

				list, ok := chooser.ChooserList().(*yarpctest.FakePeerList)
				require.True(t, ok, "list is a fake peer list")

				dispatcher := yarpc.NewDispatcher(c)
				assert.NoError(t, dispatcher.Start(), "error starting")
				assert.NoError(t, dispatcher.Stop(), "error stopping")

				_ = updater
				_ = list
			},
		},
		{
			desc: "peer chooser preset",
			given: whitespace.Expand(`
				transports:
					fake-transport:
						nop: ":1234"
				outbounds:
					their-service:
						unary:
							fake-transport:
								nop: "*.*"
								with: fake-preset
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound, ok := c.Outbounds["their-service"]
				require.True(t, ok, "config has outbound")

				require.NotNil(t, outbound.Unary, "must have unary outbound")
				unary, ok := outbound.Unary.(*yarpctest.FakeOutbound)
				require.True(t, ok, "unary outbound must be fake outbound")

				transports := unary.Transports()
				require.Equal(t, len(transports), 1, "must have one transport")

				transport, ok := transports[0].(*yarpctest.FakeTransport)
				require.True(t, ok, "must be a fake transport")
				assert.Equal(t, transport.NopOption(), ":1234", "transport configured")

				require.NotNil(t, unary.Chooser(), "must have chooser")
				chooser, ok := unary.Chooser().(*peer.BoundChooser)
				require.True(t, ok, "unary chooser must be a bound chooser")

				updater, ok := chooser.Updater().(*yarpctest.FakePeerListUpdater)
				require.True(t, ok, "updater is a fake peer list updater")

				list, ok := chooser.ChooserList().(*yarpctest.FakePeerList)
				require.True(t, ok, "list is a fake peer list")

				dispatcher := yarpc.NewDispatcher(c)
				assert.NoError(t, dispatcher.Start(), "error starting")
				assert.NoError(t, dispatcher.Stop(), "error stopping")

				_ = updater
				_ = list
			},
		},
		{
			desc: "using a peer list updater plugin",
			given: whitespace.Expand(`
				transports:
					fake-transport:
						nop: ":1234"
				outbounds:
					their-service:
						unary:
							fake-transport:
								nop: "*.*"
								fake-list:
									fake-updater:
										watch: true
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound, ok := c.Outbounds["their-service"]
				require.True(t, ok, "config has outbound")

				require.NotNil(t, outbound.Unary, "must have unary outbound")
				unary, ok := outbound.Unary.(*yarpctest.FakeOutbound)
				require.True(t, ok, "unary outbound must be fake outbound")

				transports := unary.Transports()
				require.Equal(t, len(transports), 1, "must have one transport")

				transport, ok := transports[0].(*yarpctest.FakeTransport)
				require.True(t, ok, "must be a fake transport")
				assert.Equal(t, transport.NopOption(), ":1234", "transport configured")

				require.NotNil(t, unary.Chooser(), "must have chooser")
				chooser, ok := unary.Chooser().(*peer.BoundChooser)
				require.True(t, ok, "unary chooser must be a bound chooser")

				updater, ok := chooser.Updater().(*yarpctest.FakePeerListUpdater)
				require.True(t, ok, "updater is a peer list updater")
				assert.True(t, updater.Watch(), "peer list updater configured to watch")

				list, ok := chooser.ChooserList().(*yarpctest.FakePeerList)
				require.True(t, ok, "list is a fake peer list")
				_ = list

				dispatcher := yarpc.NewDispatcher(c)
				assert.NoError(t, dispatcher.Start(), "error starting")
				assert.NoError(t, dispatcher.Stop(), "error stopping")
			},
		},
		{
			desc: "use static peers with round robin and exercise choose",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								round-robin:
									peers:
									- 127.0.0.1:8080
									- 127.0.0.1:8081
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound := c.Outbounds["their-service"]
				unary := outbound.Unary.(*yarpctest.FakeOutbound)
				transport := unary.Transports()[0].(*yarpctest.FakeTransport)
				chooser := unary.Chooser().(*peer.BoundChooser)
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

				// TODO https://github.com/yarpc/yarpc-go/issues/968
				//require.True(t, dispatcher.IsRunning(), "dispatcher is running")
				require.True(t, transport.IsRunning(), "transport is running")
				require.True(t, unary.IsRunning(), "outbound is running")
				require.True(t, list.IsRunning(), "chooser is running")
				require.True(t, binder.IsRunning(), "binder is running")

				ctx := context.Background()
				ctx, cancel := context.WithTimeout(ctx, testtime.Second)
				defer cancel()
				peer, onFinish, err := chooser.Choose(ctx, nil)
				require.NoError(t, err, "error choosing peer")
				defer onFinish(nil)

				assert.Equal(t, peer.Identifier(), "127.0.0.1:8080", "chooses first peer")
			},
		},
		{
			desc: "use round-robin chooser",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								round-robin:
									fake-updater: {}
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound := c.Outbounds["their-service"]
				unary := outbound.Unary.(*yarpctest.FakeOutbound)
				chooser := unary.Chooser().(*peer.BoundChooser)
				list, ok := chooser.ChooserList().(*roundrobin.List)
				require.True(t, ok, "use round robin")
				_ = list
			},
		},
		{
			desc: "use least-pending chooser",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								least-pending:
									fake-updater: {}
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound := c.Outbounds["their-service"]
				unary := outbound.Unary.(*yarpctest.FakeOutbound)
				chooser := unary.Chooser().(*peer.BoundChooser)
				list, ok := chooser.ChooserList().(*peerheap.List)
				require.True(t, ok, "use peer heap")
				_ = list
			},
		},
		{
			desc: "HTTP single peer implied by URL",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							http:
								url: "https://127.0.0.1/rpc"
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound, ok := c.Outbounds["their-service"]
				require.True(t, ok, "config has outbound")

				require.NotNil(t, outbound.Unary, "must have unary outbound")
				unary, ok := outbound.Unary.(*http.Outbound)
				require.True(t, ok, "unary outbound must be HTTP outbound")

				transports := unary.Transports()
				require.Equal(t, 1, len(transports), "must have one transport")

				transport, ok := transports[0].(*http.Transport)
				require.True(t, ok, "must be an HTTP transport")

				require.NotNil(t, unary.Chooser(), "must have chooser")
				chooser, ok := unary.Chooser().(*peer.Single)
				require.True(t, ok, "unary chooser must be a single peer chooser")

				dispatcher := yarpc.NewDispatcher(c)
				assert.NoError(t, dispatcher.Start(), "error starting")
				assert.NoError(t, dispatcher.Stop(), "error stopping")

				_ = transport
				_ = chooser
			},
		},
		{
			desc: "HTTP",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							http:
								url: "https://service.example.com/rpc"
								peer: "127.0.0.1"
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound, ok := c.Outbounds["their-service"]
				require.True(t, ok, "config has outbound")

				require.NotNil(t, outbound.Unary, "must have unary outbound")
				unary, ok := outbound.Unary.(*http.Outbound)
				require.True(t, ok, "unary outbound must be HTTP outbound")

				transports := unary.Transports()
				require.Equal(t, 1, len(transports), "must have one transport")

				transport, ok := transports[0].(*http.Transport)
				require.True(t, ok, "must be an HTTP transport")

				require.NotNil(t, unary.Chooser(), "must have chooser")
				chooser, ok := unary.Chooser().(*peer.Single)
				require.True(t, ok, "unary chooser must be a single peer chooser")

				dispatcher := yarpc.NewDispatcher(c)
				assert.NoError(t, dispatcher.Start(), "error starting")
				assert.NoError(t, dispatcher.Stop(), "error stopping")

				_ = transport
				_ = chooser
			},
		},
		{
			desc: "tchannel transport",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							tchannel:
								peer: 127.0.0.1:4040
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound, ok := c.Outbounds["their-service"]
				require.True(t, ok, "config has outbound")

				require.NotNil(t, outbound.Unary, "must have unary outbound")
				unary, ok := outbound.Unary.(*tchannel.Outbound)
				require.True(t, ok, "unary outbound must be TChannel outbound")

				transports := unary.Transports()
				require.Equal(t, 1, len(transports), "must have one transport")

				transport, ok := transports[0].(*tchannel.Transport)
				require.True(t, ok, "must be an TChannel transport")

				require.NotNil(t, unary.Chooser(), "must have chooser")
				chooser, ok := unary.Chooser().(*peer.Single)
				require.True(t, ok, "unary chooser must be a single peer chooser")

				dispatcher := yarpc.NewDispatcher(c)
				assert.NoError(t, dispatcher.Start(), "error starting")
				assert.NoError(t, dispatcher.Stop(), "error stopping")

				_ = transport
				_ = chooser
			},
		},
		{
			desc: "invalid peer chooser",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								bogus-list: {}
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`no recognized peer list or chooser "bogus-list"`,
				`need one of`,
				`fake-list`,
				`least-pending`,
				`round-robin`,
			},
		},
		{
			desc: "invalid peer list",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								bogus-list:
									fake-updater: {}
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`no recognized peer list or chooser "bogus-list"`,
				`need one of`,
				`fake-list`,
				`least-pending`,
				`round-robin`,
			},
		},
		{
			desc: "invalid peer chooser preset",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								with: bogus
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`no recognized peer chooser preset "bogus"`,
				`need one of`,
				`fake`,
			},
		},
		{
			desc: "invalid peer chooser decode",
			given: whitespace.Expand(`
				transports:
					fake-transport:
						nop: ":1234"
				outbounds:
					their-service:
						unary:
							fake-transport:
								nop: "*.*"
								fake-chooser:
									- 127.0.0.1:8080
									- 127.0.0.1:8081
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`failed to read attribute "fake-chooser"`,
			},
		},
		{
			desc: "invalid peer list decode",
			given: whitespace.Expand(`
				transports:
					fake-transport:
						nop: ":1234"
				outbounds:
					their-service:
						unary:
							fake-transport:
								nop: "*.*"
								fake-list:
									- 127.0.0.1:8080
									- 127.0.0.1:8081
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`failed to read attribute "fake-list"`,
			},
		},
		{
			desc: "invalid peer list updater",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								fake-list:
									bogus-updater: 10
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`no recognized peer list updater in config`,
			},
		},
		{
			desc: "too many peer list updaters",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								nop: "*.*"
								fake-list:
									fake-updater:
										- 127.0.0.1:8080
										- 127.0.0.1:8081
									invalid-updater:
										- 127.0.0.1:8080
										- 127.0.0.1:8081
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`found too many peer list updaters in config: got`,
				"fake-updater", "invalid-updater",
			},
		},
		{
			desc: "invalid peer list updater decode",
			given: whitespace.Expand(`
				transports:
					fake-transport:
						nop: ":1234"
				outbounds:
					their-service:
						unary:
							fake-transport:
								nop: "*.*"
								fake-list:
									fake-updater:
										- 127.0.0.1:8080
										- 127.0.0.1:8081
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`failed to read attribute "fake-updater"`,
			},
		},
		{
			desc: "extraneous config in combination with single peer",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								peer: a
								conspicuously: present
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`unrecognized attributes in outbound config: `,
				`conspicuously`,
				`present`,
			},
		},
		{
			desc: "extraneous transport config in combination with list config",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								conspicuously: present
								fake-list:
									peers:
										- 127.0.0.1:8080
										- 127.0.0.1:8081
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`unrecognized attributes in outbound config: `,
				`conspicuously`,
				`present`,
			},
		},
		{
			desc: "extraneous config in combination with multiple peers",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								fake-list:
									peers:
										- 127.0.0.1:8080
										- 127.0.0.1:8081
									conspicuously: present
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`has invalid keys:`,
				`conspicuously`,
			},
		},
		{
			desc: "extraneous config in combination with preset",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								with: fake-preset
								conspicuously: present
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`conspicuously`,
				`present`,
			},
		},
		{
			desc: "invalid list peers",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								fake-list:
									peers:
										host1: 127.0.0.1:8080
										host2: 127.0.0.1:8081
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`failed to read attribute "peers"`,
			},
		},
		{
			desc: "extraneous config in combination with custom updater",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								fake-list:
									fake-updater:
										watch: true
										conspicuously: present
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`conspicuously`,
			},
		},
		{
			desc: "missing peer list config",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport: {}
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`no peer list or chooser provided in config`,
				`need one of`,
				`fake-list`,
				`least-pending`,
				`round-robin`,
			},
		},
		{
			desc: "invalid peer chooser builder",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								invalid-chooser: {}
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`could not create invalid-chooser`,
			},
		},
		{
			desc: "invalid peer list builder",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								invalid-list:
									fake-updater: {}
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`could not create invalid-list`,
			},
		},
		{
			desc: "invalid peer list updater builder",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								fake-list:
									invalid-updater: {}
			`),
			wantErr: []string{
				`failed to configure unary outbound for "their-service": `,
				`could not create invalid-updater`,
			},
		},
		{
			desc: "interpolation fallback",
			given: whitespace.Expand(`
				transports:
					fake-transport:
						nop: ":${FIRST_VAR:1234}"
				outbounds:
					their-service:
						unary:
							fake-transport:
								nop: "${SECOND_VAR:*.*}"
								peer: 127.0.0.1:${THIRD_VAR:8080}
			`),
			test: func(t *testing.T, c yarpc.Config) {
				outbound, ok := c.Outbounds["their-service"]
				require.True(t, ok, "config has outbound")

				require.NotNil(t, outbound.Unary, "must have unary outbound")
				unary, ok := outbound.Unary.(*yarpctest.FakeOutbound)
				require.True(t, ok, "unary outbound must be fake outbound")
				assert.Equal(t, "*.*", unary.NopOption(), "must have configured pattern")

				transports := unary.Transports()
				require.Equal(t, 1, len(transports), "must have one transport")

				transport, ok := transports[0].(*yarpctest.FakeTransport)
				require.True(t, ok, "must be a fake transport")
				assert.Equal(t, ":1234", transport.NopOption(), "transport configured")

				require.NotNil(t, unary.Chooser(), "must have chooser")
				chooser, ok := unary.Chooser().(*peer.Single)
				require.True(t, ok, "unary chooser must be a single peer chooser")
				assert.Equal(t, "127.0.0.1:8080", chooser.Introspect().Peers[0].Identifier, "incorrect peer")

				dispatcher := yarpc.NewDispatcher(c)
				assert.NoError(t, dispatcher.Start(), "error starting")
				assert.NoError(t, dispatcher.Stop(), "error stopping")

				_ = chooser
			},
		},
		{
			desc: "interpolation to env var",
			given: whitespace.Expand(`
				transports:
					fake-transport:
						nop: ":${FIRST_VAR:1234}"
				outbounds:
					their-service:
						unary:
							fake-transport:
								nop: "${SECOND_VAR:*.*}"
								peer: 127.0.0.1:${THIRD_VAR:808-0}
			`),
			env: map[string]string{
				"FIRST_VAR":  "3456",
				"SECOND_VAR": "A*A",
				"THIRD_VAR":  "9000",
			},
			test: func(t *testing.T, c yarpc.Config) {
				outbound, ok := c.Outbounds["their-service"]
				require.True(t, ok, "config has outbound")

				require.NotNil(t, outbound.Unary, "must have unary outbound")
				unary, ok := outbound.Unary.(*yarpctest.FakeOutbound)
				require.True(t, ok, "unary outbound must be fake outbound")
				assert.Equal(t, "A*A", unary.NopOption(), "must have configured pattern")

				transports := unary.Transports()
				require.Equal(t, 1, len(transports), "must have one transport")

				transport, ok := transports[0].(*yarpctest.FakeTransport)
				require.True(t, ok, "must be a fake transport")
				assert.Equal(t, ":3456", transport.NopOption(), "transport configured")

				require.NotNil(t, unary.Chooser(), "must have chooser")
				chooser, ok := unary.Chooser().(*peer.Single)
				require.True(t, ok, "unary chooser must be a single peer chooser")
				assert.Equal(t, "127.0.0.1:9000", chooser.Introspect().Peers[0].Identifier, "incorrect peer")

				dispatcher := yarpc.NewDispatcher(c)
				assert.NoError(t, dispatcher.Start(), "error starting")
				assert.NoError(t, dispatcher.Stop(), "error stopping")

				_ = chooser
			},
		},
		{
			desc: "interpolation to env var for chooser and updater",
			given: whitespace.Expand(`
				outbounds:
					their-service:
						unary:
							fake-transport:
								fake-list:
									nop: "${LIST_VAR:list}"
									fake-updater:
										nop: "${UPDATER_VAR:updater}"
										watch: true
			`),
			env: map[string]string{
				"LIST_VAR":    "envlist",
				"UPDATER_VAR": "envupdater",
			},
			test: func(t *testing.T, c yarpc.Config) {
				outbound, ok := c.Outbounds["their-service"]
				require.True(t, ok, "config has outbound")

				require.NotNil(t, outbound.Unary, "must have unary outbound")
				unary, ok := outbound.Unary.(*yarpctest.FakeOutbound)
				require.True(t, ok, "unary outbound must be fake outbound")

				transports := unary.Transports()
				require.Equal(t, 1, len(transports), "must have one transport")

				require.NotNil(t, unary.Chooser(), "must have chooser")
				chooser, ok := unary.Chooser().(*peer.BoundChooser)
				require.True(t, ok, "unary chooser must be a bound chooser")

				updater, ok := chooser.Updater().(*yarpctest.FakePeerListUpdater)
				require.True(t, ok, "updater is a peer list updater")
				assert.True(t, updater.Watch(), "peer list updater configured to watch")
				assert.Equal(t, "envupdater", updater.Nop(), "did not properly interpolate variables for peer updater")

				list, ok := chooser.ChooserList().(*yarpctest.FakePeerList)
				require.True(t, ok, "list is a fake peer list")
				assert.Equal(t, "envlist", list.Nop(), "did not properly interpolate variables for peer list")
				_ = list

				dispatcher := yarpc.NewDispatcher(c)
				assert.NoError(t, dispatcher.Start(), "error starting")
				assert.NoError(t, dispatcher.Stop(), "error stopping")

				_ = chooser
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			configer := yarpctest.NewFakeConfigurator(yarpcconfig.InterpolationResolver(mapVariableResolver(tt.env)))
			configer.MustRegisterTransport(http.TransportSpec())
			configer.MustRegisterTransport(tchannel.TransportSpec(tchannel.Tracer(opentracing.NoopTracer{})))
			configer.MustRegisterPeerList(peerheap.Spec())
			configer.MustRegisterPeerList(roundrobin.Spec())
			configer.MustRegisterPeerChooser(invalidPeerChooserSpec())
			configer.MustRegisterPeerList(invalidPeerListSpec())
			configer.MustRegisterPeerListUpdater(invalidPeerListUpdaterSpec())

			config, err := configer.LoadConfigFromYAML("fake-service", strings.NewReader(tt.given))
			if err != nil {
				if len(tt.wantErr) > 0 {
					// Check for every required error substring
					for _, wantErr := range tt.wantErr {
						require.Contains(t, err.Error(), wantErr, "expected error")
					}
				} else {
					require.NoError(t, err, "error loading config")
				}
			} else if len(tt.wantErr) > 0 {
				require.Error(t, err, "expected error")
			}
			if tt.test != nil {
				tt.test(t, config)
			}
		})
	}
}

type invalidPeerChooserConfig struct {
}

func buildInvalidPeerChooserConfig(c *invalidPeerChooserConfig, t peerapi.Transport, kit *yarpcconfig.Kit) (peerapi.Chooser, error) {
	return nil, errors.New("could not create invalid-chooser")
}

func invalidPeerChooserSpec() yarpcconfig.PeerChooserSpec {
	return yarpcconfig.PeerChooserSpec{
		Name:             "invalid-chooser",
		BuildPeerChooser: buildInvalidPeerChooserConfig,
	}
}

type invalidPeerListConfig struct {
}

func buildInvalidPeerListConfig(c *invalidPeerListConfig, t peerapi.Transport, kit *yarpcconfig.Kit) (peerapi.ChooserList, error) {
	return nil, errors.New("could not create invalid-list")
}

func invalidPeerListSpec() yarpcconfig.PeerListSpec {
	return yarpcconfig.PeerListSpec{
		Name:          "invalid-list",
		BuildPeerList: buildInvalidPeerListConfig,
	}
}

type invalidPeerListUpdaterConfig struct {
}

func buildInvalidPeerListUpdater(c *invalidPeerListUpdaterConfig, kit *yarpcconfig.Kit) (peerapi.Binder, error) {
	return nil, errors.New("could not create invalid-updater")
}

func invalidPeerListUpdaterSpec() yarpcconfig.PeerListUpdaterSpec {
	return yarpcconfig.PeerListUpdaterSpec{
		Name:                 "invalid-updater",
		BuildPeerListUpdater: buildInvalidPeerListUpdater,
	}
}

func TestBuildPeerListInvalidKit(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// We build a fake InboundConfig that embeds the PeerList. This will let
	// us call PeerList.BuildPeerList with the wrong Kit.
	type inboundConfig struct {
		yarpcconfig.PeerChooser
	}

	configer := yarpctest.NewFakeConfigurator()
	configer.MustRegisterTransport(yarpcconfig.TransportSpec{
		Name: "foo",
		BuildTransport: func(struct{}, *yarpcconfig.Kit) (transport.Transport, error) {
			return transporttest.NewMockTransport(mockCtrl), nil
		},
		BuildInbound: func(cfg *inboundConfig, _ transport.Transport, k *yarpcconfig.Kit) (transport.Inbound, error) {
			_, err := cfg.BuildPeerChooser(peertest.NewMockTransport(mockCtrl), hostport.Identify, k)
			assert.Error(t, err, "BuildPeerList should fail with an invalid Kit")
			return transporttest.NewMockInbound(mockCtrl), err
		},
	})

	_, err := configer.LoadConfig("myservice", map[string]interface{}{
		"inbounds": map[string]interface{}{
			"foo": map[string]interface{}{"with": "irrelevant"},
		},
	})
	require.Error(t, err, "LoadConfig should fail")
	assert.Contains(t, err.Error(),
		"invalid Kit: make sure you passed in the same Kit your Build function received")
}

func mapVariableResolver(m map[string]string) interpolate.VariableResolver {
	return func(name string) (value string, ok bool) {
		value, ok = m[name]
		return
	}
}
