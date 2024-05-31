// Copyright (c) 2024 Uber Technologies, Inc.
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
	peerbind "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpcconfig"
)

// FakeTransportConfig configures the FakeTransport.
type FakeTransportConfig struct {
	Nop string `config:"nop,interpolate"`
}

func buildFakeTransport(c *FakeTransportConfig, kit *yarpcconfig.Kit) (transport.Transport, error) {
	return NewFakeTransport(NopTransportOption(c.Nop)), nil
}

// FakeOutboundConfig configures the FakeOutbound.
type FakeOutboundConfig struct {
	yarpcconfig.PeerChooser

	Nop string `config:"nop,interpolate"`
}

func buildFakeUnaryOutbound(c *FakeOutboundConfig, t transport.Transport, kit *yarpcconfig.Kit) (transport.UnaryOutbound, error) {
	return buildFakeOutbound(c, t, kit)
}

func buildFakeOnewayOutbound(c *FakeOutboundConfig, t transport.Transport, kit *yarpcconfig.Kit) (transport.OnewayOutbound, error) {
	return buildFakeOutbound(c, t, kit)
}

func buildFakeStreamOutbound(c *FakeOutboundConfig, t transport.Transport, kit *yarpcconfig.Kit) (transport.StreamOutbound, error) {
	return buildFakeOutbound(c, t, kit)
}

func buildFakeOutbound(c *FakeOutboundConfig, t transport.Transport, kit *yarpcconfig.Kit) (*FakeOutbound, error) {
	x := t.(*FakeTransport)
	chooser, err := c.BuildPeerChooser(x, hostport.Identify, kit)
	if err != nil {
		return nil, err
	}
	return x.NewOutbound(chooser, NopOutboundOption(c.Nop)), nil
}

// FakeTransportSpec returns a configurator spec for the fake-transport
// transport type, suitable for passing to Configurator.MustRegisterTransport.
func FakeTransportSpec() yarpcconfig.TransportSpec {
	return yarpcconfig.TransportSpec{
		Name:                "fake-transport",
		BuildTransport:      buildFakeTransport,
		BuildUnaryOutbound:  buildFakeUnaryOutbound,
		BuildOnewayOutbound: buildFakeOnewayOutbound,
		BuildStreamOutbound: buildFakeStreamOutbound,
		PeerChooserPresets: []yarpcconfig.PeerChooserPreset{
			FakePeerChooserPreset(),
		},
	}
}

// FakePeerChooserConfig configures the FakePeerChooser.
type FakePeerChooserConfig struct {
	Nop string `config:"nop,interpolate"`
}

func buildFakePeerChooser(c *FakePeerChooserConfig, t peer.Transport, kit *yarpcconfig.Kit) (peer.Chooser, error) {
	return NewFakePeerChooser(ChooserNop(c.Nop)), nil
}

// FakePeerChooserSpec returns a configurator spec for the fake-chooser FakePeerChooser
// peer selection strategy, suitable for passing to
// Configurator.MustRegisterPeerChooser.
func FakePeerChooserSpec() yarpcconfig.PeerChooserSpec {
	return yarpcconfig.PeerChooserSpec{
		Name:             "fake-chooser",
		BuildPeerChooser: buildFakePeerChooser,
	}
}

// FakePeerListConfig configures the FakePeerList.
type FakePeerListConfig struct {
	Nop string `config:"nop,interpolate"`
}

func buildFakePeerList(c *FakePeerListConfig, t peer.Transport, kit *yarpcconfig.Kit) (peer.ChooserList, error) {
	return NewFakePeerList(ListNop(c.Nop)), nil
}

// FakePeerListSpec returns a configurator spec for the fake-list FakePeerList
// peer selection strategy, suitable for passing to
// Configurator.MustRegisterPeerList.
func FakePeerListSpec() yarpcconfig.PeerListSpec {
	return yarpcconfig.PeerListSpec{
		Name:          "fake-list",
		BuildPeerList: buildFakePeerList,
	}
}

// FakePeerListUpdaterConfig configures a fake-updater FakePeerListUpdater.
// It has a fake "watch" property that adds the Watch option for
// NewFakePeerListUpdater when you build a peer list with this config.
type FakePeerListUpdaterConfig struct {
	FakeUpdater string `config:"fake-updater"`
	Nop         string `config:"nop,interpolate"`
	Watch       bool   `config:"watch"`
}

func buildFakePeerListUpdater(c *FakePeerListUpdaterConfig, kit *yarpcconfig.Kit) (peer.Binder, error) {
	var opts []FakePeerListUpdaterOption
	if c.Watch {
		opts = append(opts, Watch)
	}
	if c.Nop != "" {
		opts = append(opts, UpdaterNop(c.Nop))
	}
	return func(pl peer.List) transport.Lifecycle {
		return NewFakePeerListUpdater(opts...)
	}, nil
}

// FakePeerListUpdaterSpec returns a configurator spec for the fake-updater
// FakePeerListUpdater type, suitable for passing to Configurator.MustRegisterPeerListUpdaterSpec.
func FakePeerListUpdaterSpec() yarpcconfig.PeerListUpdaterSpec {
	return yarpcconfig.PeerListUpdaterSpec{
		Name:                 "fake-updater",
		BuildPeerListUpdater: buildFakePeerListUpdater,
	}
}

// NewFakeConfigurator returns a configurator with fake-transport,
// fake-peer-list, and fake-peer-list-updater specs already registered,
// suitable for testing the configurator.
func NewFakeConfigurator(opts ...yarpcconfig.Option) *yarpcconfig.Configurator {
	configurator := yarpcconfig.New(opts...)
	configurator.MustRegisterTransport(FakeTransportSpec())
	configurator.MustRegisterPeerChooser(FakePeerChooserSpec())
	configurator.MustRegisterPeerList(FakePeerListSpec())
	configurator.MustRegisterPeerListUpdater(FakePeerListUpdaterSpec())
	return configurator
}

// FakePeerChooserPreset is a PeerChooserPreset which builds a FakePeerList buind to
// a FakePeerListUpdater.
func FakePeerChooserPreset() yarpcconfig.PeerChooserPreset {
	return yarpcconfig.PeerChooserPreset{
		Name: "fake-preset",
		BuildPeerChooser: func(peer.Transport, *yarpcconfig.Kit) (peer.Chooser, error) {
			return peerbind.Bind(
				NewFakePeerList(), func(peer.List) transport.Lifecycle {
					return NewFakePeerListUpdater()
				}), nil
		},
	}
}
