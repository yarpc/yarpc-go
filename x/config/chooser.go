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
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/yarpc/api/peer"
	peerbind "go.uber.org/yarpc/peer"
)

// PeerList facilitates decoding and building peer choosers.
// The peer chooser combines a peer list (for the peer selection strategy, like
// least-pending or round-robin) with a peer list binder (like static peers or
// dynamic peers from DNS or watching a file in a particular format).
type PeerList struct {
	peerList
}

// peerList is the private representation of PeerList that captures
// decoded configuration without revealing it on the public type.
type peerList struct {
	Peer string       `config:"peer,interpolate"`
	Etc  attributeMap `config:",squash"`
}

// Empty returns whether the peer list configuration is empty.
// This is a facility for the HTTP transport specifically since it can infer
// the configuration for the single-peer case from its "url" attribute.
func (pc PeerList) Empty() bool {
	c := pc.peerList
	return c.Peer == "" && len(c.Etc) == 0
}

// BuildPeerList translates a chooser configuration into a peer chooser, backed
// by a peer list bound to a peer list binder.
func (pc PeerList) BuildPeerList(transport peer.Transport, identify func(string) peer.Identifier, kit *Kit) (peer.Chooser, error) {
	c := pc.peerList
	// Establish a peer selection strategy.

	// Special case for single-peer outbounds.
	if c.Peer != "" {
		if len(c.Etc) > 0 {
			return nil, fmt.Errorf("unrecognized attributes in outbound config: %+v", c.Etc)
		}

		return peerbind.NewSingle(identify(c.Peer), transport), nil
	}

	// All multi-peer choosers may combine a peer list (for sharding or
	// load-balancing) and a peer list updater.

	// Find a property name that corresponds to a peer chooser/list and construct it.
	for peerListName := range c.Etc {
		peerListSpec := kit.peerListSpec(peerListName)
		if peerListSpec == nil {
			continue
		}

		var peerListConfig attributeMap
		if _, err := c.Etc.Pop(peerListName, &peerListConfig); err != nil {
			return nil, err
		}

		if len(c.Etc) > 0 {
			return nil, fmt.Errorf("unrecognized attributes in outbound config: %+v", c.Etc)
		}

		peerListUpdater, err := buildPeerListUpdater(peerListConfig, identify, kit)
		if err != nil {
			return nil, err
		}

		chooserBuilder, err := peerListSpec.PeerList.Decode(peerListConfig)
		if err != nil {
			return nil, err
		}

		result, err := chooserBuilder.Build(transport, kit)
		if err != nil {
			return nil, err
		}

		peerList := result.(peer.ChooserList)

		return peerbind.Bind(peerList, peerListUpdater), nil
	}

	msg := fmt.Sprintf(
		"no recognized peer list in config: got %s", strings.Join(c.names(), ", "))
	if available := kit.peerListSpecNames(); len(available) > 0 {
		msg = fmt.Sprintf("%s; need one of %s", msg, strings.Join(available, ", "))
	}
	return nil, errors.New(msg)
}

func (c peerList) names() (names []string) {
	for name := range c.Etc {
		names = append(names, name)
	}
	sort.Strings(names)
	return
}

func buildPeerListUpdater(c attributeMap, identify func(string) peer.Identifier, kit *Kit) (peer.Binder, error) {
	var peers []string
	_, err := c.Pop("peers", &peers)
	if err != nil {
		return nil, err
	}

	if len(peers) > 0 {
		return peerbind.BindPeers(identifyAll(identify, peers)), nil
	}

	for peerListUpdaterName := range c {
		peerListUpdaterSpec := kit.peerListUpdaterSpec(peerListUpdaterName)
		if peerListUpdaterSpec == nil {
			continue
		}

		var peerListUpdaterConfig attributeMap
		if _, err := c.Pop(peerListUpdaterName, &peerListUpdaterConfig); err != nil {
			return nil, err
		}

		// This decodes all attributes on the peer list updater block, including
		// the field with the name of the peer list updater.
		peerListUpdaterBuilder, err := peerListUpdaterSpec.PeerListUpdater.Decode(peerListUpdaterConfig)
		if err != nil {
			return nil, err
		}

		result, err := peerListUpdaterBuilder.Build(kit)
		if err != nil {
			return nil, err
		}

		binder := result.(peer.Binder)

		return binder, nil
	}

	return nil, fmt.Errorf(
		"no recognized peer list updater in config: got %s; need one of %s",
		strings.Join(configNames(c), ", "),
		strings.Join(kit.peerListUpdaterSpecNames(), ", "),
	)
}

func configNames(c attributeMap) (names []string) {
	for name := range c {
		names = append(names, name)
	}
	sort.Strings(names)
	return
}

func identifyAll(identify func(string) peer.Identifier, peers []string) []peer.Identifier {
	pids := make([]peer.Identifier, len(peers))
	for i, peer := range peers {
		pids[i] = identify(peer)
	}
	return pids
}
