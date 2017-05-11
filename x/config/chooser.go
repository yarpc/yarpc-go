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
	"sort"
	"strings"

	"go.uber.org/yarpc/api/peer"
	peerbind "go.uber.org/yarpc/peer"
)

// PeerChooser facilitates decoding and building peer choosers.
// The peer chooser combines a peer list (for the peer selection strategy, like
// least-pending or round-robin) with a peer list binder (like static peers or
// dynamic peers from DNS or watching a file in a particular format).
type PeerChooser struct {
	peerChooser
}

// peerChooser is the private representation of PeerChooser that captures
// decoded configuration without revealing it on the public type.
type peerChooser struct {
	Peer   string       `config:"peer,interpolate"`
	Preset string       `config:"with,interpolate"`
	Etc    attributeMap `config:",squash"`
}

// Empty returns whether the peer list configuration is empty.
// This is a facility for the HTTP transport specifically since it can infer
// the configuration for the single-peer case from its "url" attribute.
func (pc PeerChooser) Empty() bool {
	c := pc.peerChooser
	return c.Peer == "" && c.Preset == "" && len(c.Etc) == 0
}

// BuildPeerChooser translates the decoded configuration into a peer.Chooser.
func (pc PeerChooser) BuildPeerChooser(transport peer.Transport, identify func(string) peer.Identifier, kit *Kit) (peer.Chooser, error) {
	c := pc.peerChooser
	// Establish a peer selection strategy.
	switch {
	case c.Peer != "":
		// myoutbound:
		//   outboundopt1: ...
		//   outboundopt2: ...
		//   peer: 127.0.0.1:8080
		if len(c.Etc) > 0 {
			return nil, fmt.Errorf("unrecognized attributes in outbound config: %+v", c.Etc)
		}
		return peerbind.NewSingle(identify(c.Peer), transport), nil
	case c.Preset != "":
		// myoutbound:
		//   outboundopt1: ...
		//   outboundopt2: ...
		//   with: somepreset
		if len(c.Etc) > 0 {
			return nil, fmt.Errorf("unrecognized attributes in outbound config: %+v", c.Etc)
		}

		preset, err := kit.peerChooserPreset(pc.Preset)
		if err != nil {
			return nil, err
		}

		return preset.Build(transport, kit)
	default:
		// myoutbound:
		//   outboundopt1: ...
		//   outboundopt2: ...
		//   my-peer-list:
		//     ...
		return pc.buildPeerChooser(transport, identify, kit)
	}
}

func (pc PeerChooser) buildPeerChooser(transport peer.Transport, identify func(string) peer.Identifier, kit *Kit) (peer.Chooser, error) {
	peerListName, peerListConfig, err := getPeerListInfo(pc.Etc, kit)
	if err != nil {
		return nil, err
	}

	peerListSpec, err := kit.peerListSpec(peerListName)
	if err != nil {
		return nil, err
	}

	// This builds the peer list updater and also removes its entry from the
	// map. Given,
	//
	//   least-pending:
	//     failurePenalty: 5s
	//     dns:
	//       ..
	//
	// We will be left with only failurePenalty in the map so that we can simply
	// decode it into the peer list configuration type.
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
	peerChooser := result.(peer.ChooserList)

	return peerbind.Bind(peerChooser, peerListUpdater), nil
}

// getPeerListInfo extracts the peer list entry from the given attribute map. It
// must be the only remaining entry.
//
// For example, in
//
//   myoutbound:
//     outboundopt1: ...
//     outboundopt2: ...
//     my-peer-list:
//       ...
//
// By the time getPeerListInfo is called, the map must only be,
//
//   my-peer-list:
//     ...
//
// The name of the peer list (my-peer-list) is returned with the attributes
// specified under that entry.
func getPeerListInfo(etc attributeMap, kit *Kit) (name string, config attributeMap, err error) {
	names := etc.Keys()
	switch len(names) {
	case 0:
		err = fmt.Errorf("no peer list provided in config, need one of: %+v", kit.peerListSpecNames())
	default:
		err = fmt.Errorf("unrecognized attributes in outbound config: %+v", etc)
	case 1:
		name = names[0]
		_, err = etc.Pop(name, &config)
	}
	return
}

// buildPeerListUpdater builds the peer list updater given the peer list
// configuration map. For example, we might get,
//
//   least-pending:
//     failurePenalty: 5s
//     dns:
//       name: myservice.example.com
//       record: A
func buildPeerListUpdater(c attributeMap, identify func(string) peer.Identifier, kit *Kit) (peer.Binder, error) {
	// Special case for explicit list of peers.
	var peers []string
	if _, err := c.Pop("peers", &peers); err != nil {
		return nil, err
	}
	if len(peers) > 0 {
		return peerbind.BindPeers(identifyAll(identify, peers)), nil
	}
	// TODO: Make peers a separate peer list updater that is registered by
	// default instead of special casing here.

	var (
		// The peer list updater config is in the same namespace as the
		// attributes for the peer list config. We want to ensure that there is
		// exactly one peer list updater in the config.
		foundUpdaters []string

		// The peer list updater spec we'll actually use.
		peerListUpdaterSpec *compiledPeerListUpdaterSpec
	)

	for name := range c {
		spec := kit.peerListUpdaterSpec(name)
		if spec != nil {
			peerListUpdaterSpec = spec
			foundUpdaters = append(foundUpdaters, name)
		}
	}

	switch len(foundUpdaters) {
	case 0:
		return nil, fmt.Errorf(
			"no recognized peer list updater in config: got %s; need one of %s",
			strings.Join(configNames(c), ", "),
			strings.Join(kit.peerListUpdaterSpecNames(), ", "),
		)
	default:
		sort.Strings(foundUpdaters) // deterministic error message
		return nil, fmt.Errorf(
			"found too many peer list updaters in config: got %s",
			strings.Join(foundUpdaters, ", "))
	case 1:
		// fall through to logic below
	}

	var peerListUpdaterConfig attributeMap
	if _, err := c.Pop(foundUpdaters[0], &peerListUpdaterConfig); err != nil {
		return nil, err
	}

	// This decodes all attributes on the peer list updater block, including the
	// field with the name of the peer list updater.
	peerListUpdaterBuilder, err := peerListUpdaterSpec.PeerListUpdater.Decode(peerListUpdaterConfig)
	if err != nil {
		return nil, err
	}

	result, err := peerListUpdaterBuilder.Build(kit)
	if err != nil {
		return nil, err
	}

	return result.(peer.Binder), nil
}

func identifyAll(identify func(string) peer.Identifier, peers []string) []peer.Identifier {
	pids := make([]peer.Identifier, len(peers))
	for i, p := range peers {
		pids[i] = identify(p)
	}
	return pids
}

func configNames(c attributeMap) (names []string) {
	for name := range c {
		names = append(names, name)
	}
	sort.Strings(names)
	return
}
