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

// This file provides fake implementations of most YARPC building blocks for
// the purpose of testing configuration using custom transports, choosers, and
// binders..

import (
	"context"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	intsync "go.uber.org/yarpc/internal/sync"
	"go.uber.org/yarpc/peer/hostport"
)

func newFakeTransport(address string) *fakeTransport {
	return &fakeTransport{
		Lifecycle: intsync.NewNopLifecycle(),
		address:   address,
	}
}

type fakeTransport struct {
	transport.Lifecycle
	address string
}

func (t *fakeTransport) NewOutbound(c peer.Chooser) transport.UnaryOutbound {
	return &fakeOutbound{
		once:      intsync.Once(),
		transport: t,
		chooser:   c,
	}
}

type fakePeer struct {
	id hostport.PeerIdentifier
}

func (p *fakePeer) Identifier() string {
	return string(p.id)
}

func (p *fakePeer) Status() peer.Status {
	return peer.Status{
		ConnectionStatus:    peer.Available,
		PendingRequestCount: 0,
	}
}

func (p *fakePeer) StartRequest() {
}

func (p *fakePeer) EndRequest() {
}

func (t *fakeTransport) RetainPeer(id peer.Identifier, ps peer.Subscriber) (peer.Peer, error) {
	peer := &fakePeer{id: id.(hostport.PeerIdentifier)}
	return peer, nil
}

func (t *fakeTransport) ReleasePeer(id peer.Identifier, ps peer.Subscriber) error {
	return nil
}

type fakeTransportConfig struct {
	Address string `config:"address"`
}

type fakeOutbound struct {
	once      intsync.LifecycleOnce
	transport *fakeTransport
	chooser   peer.Chooser
}

func (o *fakeOutbound) Start() error {
	return o.once.Start(func() error {
		return o.chooser.Start()
	})
}

func (o *fakeOutbound) Stop() error {
	return o.once.Stop(func() error {
		return o.chooser.Stop()
	})
}

func (o *fakeOutbound) IsRunning() bool {
	return o.once.IsRunning()
}

func (o *fakeOutbound) Transports() []transport.Transport {
	return []transport.Transport{o.transport}
}

func (o *fakeOutbound) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	return nil, nil
}

type fakeOutboundConfig struct {
	ChooserConfig
}

type fakeChooser struct {
	transport.Lifecycle
}

func newFakeChooser() *fakeChooser {
	return &fakeChooser{
		Lifecycle: intsync.NewNopLifecycle(),
	}
}

func buildFakeChooser() peer.ChooserList {
	return newFakeChooser()
}

func (c *fakeChooser) Choose(ctx context.Context, req *transport.Request) (peer.Peer, func(error), error) {
	return nil, nil, nil
}

func (c *fakeChooser) Update(up peer.ListUpdates) error {
	return nil
}

func fakeChooserSpec() ChooserSpec {
	return ChooserSpec{
		Name:       "fakechooser",
		NewChooser: buildFakeChooser,
	}
}

type fakePeerListUpdater struct {
	transport.Lifecycle
	watch bool
}

func newFakeBinder(watch bool) *fakePeerListUpdater {
	return &fakePeerListUpdater{
		Lifecycle: intsync.NewNopLifecycle(),
		watch:     watch,
	}
}

type fakePeerListUpdaterConfig struct {
	Watch bool `config:"watch"`
}

func buildFakeBinder(c *fakePeerListUpdaterConfig, kit *Kit) (peer.Binder, error) {
	return func(pl peer.List) transport.Lifecycle {
		return newFakeBinder(c.Watch)
	}, nil
}

func fakePeerListUpdaterSpec() BinderSpec {
	return BinderSpec{
		Name:        "fakebinder",
		BuildBinder: buildFakeBinder,
	}
}

func buildFakeTransport(c *fakeTransportConfig, kit *Kit) (transport.Transport, error) {
	return newFakeTransport(c.Address), nil
}

func buildFakeOutbound(c *fakeOutboundConfig, t transport.Transport, kit *Kit) (transport.UnaryOutbound, error) {
	x := t.(*fakeTransport)
	chooser, err := c.ChooserConfig.BuildChooser(x, hostport.Identify, kit)
	if err != nil {
		return nil, err
	}
	return x.NewOutbound(chooser), nil
}

func fakeTransportSpec() TransportSpec {
	return TransportSpec{
		Name:               "faketransport",
		BuildTransport:     buildFakeTransport,
		BuildUnaryOutbound: buildFakeOutbound,
	}
}

func fakeConfigurator() *Configurator {
	configurator := New()
	configurator.MustRegisterTransport(fakeTransportSpec())
	configurator.MustRegisterChooser(fakeChooserSpec())
	configurator.MustRegisterBinder(fakePeerListUpdaterSpec())
	return configurator
}
