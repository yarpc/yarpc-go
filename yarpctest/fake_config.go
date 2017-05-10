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

package yarpctest

import (
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/x/config"
)

// FakeTransportConfig configures the FakeTransport.
type FakeTransportConfig struct {
	Nop string `config:"nop"`
}

func buildFakeTransport(c *FakeTransportConfig, kit *config.Kit) (transport.Transport, error) {
	return NewFakeTransport(NopTransportOption(c.Nop)), nil
}

// FakeOutboundConfig configures the FakeOutbound.
type FakeOutboundConfig struct {
	config.PeerList

	Nop string `config:"nop"`
}

func buildFakeOutbound(c *FakeOutboundConfig, t transport.Transport, kit *config.Kit) (transport.UnaryOutbound, error) {
	x := t.(*FakeTransport)
	chooser, err := c.PeerList.BuildPeerList(x, hostport.Identify, kit)
	if err != nil {
		return nil, err
	}
	return x.NewOutbound(chooser, NopOutboundOption(c.Nop)), nil
}

// FakeTransportSpec returns a configurator spec for the fake-transport
// transport type, suitable for passing to Configurator.MustRegisterTransport.
func FakeTransportSpec() config.TransportSpec {
	return config.TransportSpec{
		Name:               "fake-transport",
		BuildTransport:     buildFakeTransport,
		BuildUnaryOutbound: buildFakeOutbound,
	}
}

// FakePeerListConfig configures the FakePeerList.
type FakePeerListConfig struct {
}

func buildFakePeerList(c *FakePeerListConfig, t peer.Transport, kit *config.Kit) (peer.ChooserList, error) {
	return NewFakePeerList(), nil
}

// FakePeerListSpec returns a configurator spec for the fake-list FakePeerList
// peer selection strategy, suitable for passing to
// Configurator.MustRegisterPeerList.
func FakePeerListSpec() config.PeerListSpec {
	return config.PeerListSpec{
		Name:          "fake-list",
		BuildPeerList: buildFakePeerList,
	}
}

// FakePeerListUpdaterConfig configures a fake-updater FakePeerListUpdater.
// It has a fake "watch" property that adds the Watch option for
// NewFakePeerListUpdater when you build a peer list with this config.
type FakePeerListUpdaterConfig struct {
	FakeUpdater string `config:"fake-updater"`
	Watch       bool   `config:"watch"`
}

func buildFakePeerListUpdater(c *FakePeerListUpdaterConfig, kit *config.Kit) (peer.Binder, error) {
	var opts []FakePeerListUpdaterOption
	if c.Watch {
		opts = append(opts, Watch)
	}
	return func(pl peer.List) transport.Lifecycle {
		return NewFakePeerListUpdater(opts...)
	}, nil
}

// FakePeerListUpdaterSpec returns a configurator spec for the fake-updater
// FakePeerListUpdater type, suitable for passing to Configurator.MustRegisterPeerListUpdaterSpec.
func FakePeerListUpdaterSpec() config.PeerListUpdaterSpec {
	return config.PeerListUpdaterSpec{
		Name:                 "fake-updater",
		BuildPeerListUpdater: buildFakePeerListUpdater,
	}
}

// NewFakeConfigurator returns a configurator with fake-transport,
// fake-peer-list, and fake-peer-list-updater specs already registered,
// suitable for testing the configurator.
func NewFakeConfigurator() *config.Configurator {
	configurator := config.New()
	configurator.MustRegisterTransport(FakeTransportSpec())
	configurator.MustRegisterPeerList(FakePeerListSpec())
	configurator.MustRegisterPeerListUpdater(FakePeerListUpdaterSpec())
	return configurator
}
