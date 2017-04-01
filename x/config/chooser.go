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
	"fmt"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/internal/decode"
	peerbind "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/x/peerheap"
	"go.uber.org/yarpc/peer/x/roundrobin"
)

// ChooserConfig facilitates decoding and building peer choosers.
// The peer chooser combines a peer list (for the peer selection strategy, like
// least-pending or round-robin) with a peer list binder (like static peers or
// dynamic peers from DNS or watching a file in a particular format).
type ChooserConfig struct {
	With   string   `config:"with"`   // like peers / file / dns / dns-srv
	Choose string   `config:"choose"` // like round-robin / least-pending
	Peer   string   `config:"peer"`   // implies with: peer
	Peers  []string `config:"peers"`  // implies with: peers
	Etc    attributeMap
}

// Decode captures a configuration on this ChooserConfig struct.
func (c *ChooserConfig) Decode(into decode.Into) error {
	var err error

	err = into(&c.Etc)
	if err != nil {
		return fmt.Errorf(`could not decode attributes of outbound peer list chooser and updater config: %v`, err)
	}

	c.With, err = c.Etc.PopString("with")
	if err != nil {
		return fmt.Errorf(`could not decode outbound peer list updater config, "with": %v`, err)
	}

	c.Choose, err = c.Etc.PopString("choose")
	if err != nil {
		return fmt.Errorf(`could not decode outbound peer list/chooser config, "choose": %v`, err)
	}

	c.Peer, err = c.Etc.PopString("peer")
	if err != nil {
		return fmt.Errorf(`could not decode outbound "peer" config: %v`, err)
	}

	_, err = c.Etc.Pop("peers", &c.Peers)
	if err != nil {
		return fmt.Errorf(`could not decode outbound "peers" config: %v`, err)
	}

	return nil
}

// BuildChooser translates a chooser configuration into a peer chooser, backed
// by a peer list bound to a peer list updater.
func (c ChooserConfig) BuildChooser(transport peer.Transport, identify func(string) peer.Identifier, kit *Kit) (peer.Chooser, error) {
	// Establish a peer selection strategy.

	// Special case for single-peer outbounds.
	if c.Peer != "" {
		if c.With != "" {
			return nil, fmt.Errorf(`cannot combine "peer": %q and "with": %q`, c.Peer, c.With)
		}

		if c.Choose != "" {
			return nil, fmt.Errorf(`cannot combine "peer": %q and "choose": %q`, c.Peer, c.Choose)
		}

		if len(c.Peers) > 0 {
			return nil, fmt.Errorf(`cannot combine "peer": %q and "peers": %q`, c.Peer, c.Peers)
		}

		if len(c.Etc) > 0 {
			return nil, fmt.Errorf(`unrecognized attributes for outbound peer list/chooser config: %v`, c.Etc)
		}

		return peerbind.NewSingle(identify(c.Peer), transport), nil
	}

	// All multi-peer lists may combine a peer chooser/list (for sharding or
	// load-balancing) and updater (a static or dynamic source of peer
	// addresses).

	// Build a peer list for a peer selection strategy, like round-robin or
	// least-pending.
	// Determined by "choose"
	list, err := c.BuildList(transport, kit)
	if err != nil {
		return nil, err
	}

	// Build a peer list updater, using "peer", "peers", or "with" a custom
	// peer updater.
	binder, err := c.BuildBinder(transport, identify, kit)
	if err != nil {
		return nil, err
	}

	return peerbind.Bind(list, binder), nil
}

// BuildList translates a peer chooser configuration to a peer list / peer
// chooser, suitable for a peer selection strategy.  The list will be empty
// until updated with a peer list updater.
func (c ChooserConfig) BuildList(transport peer.Transport, kit *Kit) (peer.ChooserList, error) {
	switch c.Choose {
	case "":
		fallthrough
	case "least-pending":
		return peerheap.New(transport), nil
	case "round-robin":
		return roundrobin.New(transport), nil
	}

	// Look up and use a custom chooser on the configurator kit.
	chooserSpec := kit.chooser(c.Choose)
	if chooserSpec == nil {
		return nil, fmt.Errorf(`not a supported peer list/chooser %q`, c.Choose)
	}

	return chooserSpec.NewChooser(), nil
}

// BuildBinder translates a chooser configuration to a peer list binder.
// The binder is responsible for binding a peer list with a peer list updater.
func (c ChooserConfig) BuildBinder(transport peer.Transport, identify func(string) peer.Identifier, kit *Kit) (peer.Binder, error) {
	// Establish peers to bind.

	if len(c.Peers) > 0 {
		if c.With != "" {
			return nil, fmt.Errorf(`cannot combine "peers" and "with": %q`, c.With)
		}

		if len(c.Etc) > 0 {
			return nil, fmt.Errorf(`unrecognized attributes for outbound peer list/chooser config: %v`, c.Etc)
		}

		return peerbind.BindPeers(identifyAll(identify, c.Peers)), nil
	}

	// Look up and use a custom binder on the configurator kit.

	if c.With == "" {
		return nil, fmt.Errorf(`missing "peer", "peers", or "with" peer list binder`)
	}

	binderSpec := kit.binder(c.With)
	if binderSpec == nil {
		return nil, fmt.Errorf(`not a supported peer list binder %q`, c.With)
	}

	binderBuilder, err := binderSpec.Binder.Decode(c.Etc)
	if err != nil {
		return nil, fmt.Errorf(`unrecognized attributes for outbound peer list/chooser config: %v`, err)
	}

	result, err := binderBuilder.Build(kit)
	if err != nil {
		return nil, err
	}

	return result.(peer.Binder), nil
}

func identifyAll(identify func(string) peer.Identifier, peers []string) []peer.Identifier {
	pids := make([]peer.Identifier, len(peers))
	for i, peer := range peers {
		pids[i] = identify(peer)
	}
	return pids
}
