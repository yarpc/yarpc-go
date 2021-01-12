// Copyright (c) 2021 Uber Technologies, Inc.
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

package yarpcconfig

import (
	"fmt"
	"sort"
	"strings"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/internal/config"
	peerbind "go.uber.org/yarpc/peer"
)

// PeerChooser facilitates decoding and building peer choosers. A peer chooser
// combines a peer list (which implements the peer selection strategy) and a
// peer list updater (which informs the peer list about different peers),
// allowing transports to rely on these two pieces for peer selection and load
// balancing.
//
// Format
//
// Peer chooser configuration may define only one of the following keys:
// `peer`, `with`, or the name of any registered PeerListSpec.
//
// `peer` indicates that requests must be sent to a single peer.
//
// 	http:
// 	  peer: 127.0.0.1:8080
//
// Note that how this string is interpreted is transport-dependent.
//
// `with` specifies that a named peer chooser preset defined by the transport
// should be used rather than defining one by hand in the configuration.
//
// 	# Given a dev-proxy preset on your TransportSpec,
// 	http:
// 	  with: dev-proxy
//
// If the name of a registered PeerListSpec is the key, an object specifying
// the configuration parameters for the PeerListSpec is expected along with
// the name of a known peer list updater and its configuration.
//
// 	# cfg.RegisterPeerList(roundrobin.Spec())
// 	round-robin:
// 	  peers:
// 	    - 127.0.0.1:8080
// 	    - 127.0.0.1:8081
//
// In the example above, there are no configuration parameters for the round
// robin peer list. The only remaining key is the name of the peer list
// updater: `peers` which is just a static list of peers.
//
// Integration
//
// To integrate peer choosers with your transport, embed this struct into your
// outbound configuration.
//
// 	type myOutboundConfig struct {
// 		config.PeerChooser
//
// 		Address string
// 	}
//
// Then in your Build*Outbound function, use the PeerChooser.BuildPeerChooser
// method to retrieve a peer chooser for your outbound. The following example
// only works if your transport implements the peer.Transport interface.
//
// 	func buildOutbound(cfg *myOutboundConfig, t transport.Transport, k *config.Kit) (transport.UnaryOutbound, error) {
// 		myTransport := t.(*MyTransport)
// 		peerChooser, err := cfg.BuildPeerChooser(myTransport, hostport.Identify, k)
// 		if err != nil {
// 			return nil, err
// 		}
// 		return myTransport.NewOutbound(peerChooser), nil
// 	}
//
// The *config.Kit received by the Build*Outbound function MUST be passed to
// the BuildPeerChooser function as-is.
//
// Note that the keys for the PeerChooser: peer, with, and any peer list name,
// share the namespace with the attributes of your outbound configuration.
type PeerChooser struct {
	peerChooser
}

// peerChooser is the private representation of PeerChooser that captures
// decoded configuration without revealing it on the public type.
type peerChooser struct {
	Peer   string              `config:"peer,interpolate"`
	Preset string              `config:"with,interpolate"`
	Etc    config.AttributeMap `config:",squash"`
}

// Empty returns true if the PeerChooser is empty, i.e., it does not have any
// keys defined.
//
// This allows Build*Outbound functions to handle the case where the peer
// configuration is specified in a different way than the standard peer
// configuration.
func (pc PeerChooser) Empty() bool {
	return pc.Peer == "" && pc.Preset == "" && len(pc.Etc) == 0
}

// BuildPeerChooser translates the decoded configuration into a peer.Chooser.
//
// The identify function informs us how to convert string-based peer names
// into peer identifiers for the transport.
//
// The Kit received by the Build*Outbound function MUST be passed to
// BuildPeerChooser as-is.
func (pc PeerChooser) BuildPeerChooser(transport peer.Transport, identify func(string) peer.Identifier, kit *Kit) (peer.Chooser, error) {
	// Establish a peer selection strategy.
	switch {
	case pc.Peer != "":
		// myoutbound:
		//   outboundopt1: ...
		//   outboundopt2: ...
		//   peer: 127.0.0.1:8080
		if len(pc.Etc) > 0 {
			return nil, fmt.Errorf("unrecognized attributes in outbound config: %+v", pc.Etc)
		}
		return peerbind.NewSingle(identify(pc.Peer), transport), nil
	case pc.Preset != "":
		// myoutbound:
		//   outboundopt1: ...
		//   outboundopt2: ...
		//   with: somepreset
		if len(pc.Etc) > 0 {
			return nil, fmt.Errorf("unrecognized attributes in outbound config: %+v", pc.Etc)
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
	peerChooserName, peerChooserConfig, err := getPeerListInfo(pc.Etc, kit)
	if err != nil {
		return nil, err
	}

	if peerChooserSpec := kit.maybePeerChooserSpec(peerChooserName); peerChooserSpec != nil {
		chooserBuilder, err := peerChooserSpec.PeerChooser.Decode(peerChooserConfig, config.InterpolateWith(kit.resolver))
		if err != nil {
			return nil, err
		}
		result, err := chooserBuilder.Build(transport, kit)
		if err != nil {
			return nil, err
		}
		return result.(peer.Chooser), nil
	}

	// if there was no chooser registered, we assume we have a peer list registered
	peerListSpec, err := kit.peerListSpec(peerChooserName)
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
	peerListUpdater, err := buildPeerListUpdater(peerChooserConfig, identify, kit)
	if err != nil {
		return nil, err
	}

	listBuilder, err := peerListSpec.PeerList.Decode(peerChooserConfig, config.InterpolateWith(kit.resolver))
	if err != nil {
		return nil, err
	}
	result, err := listBuilder.Build(transport, kit)
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
func getPeerListInfo(etc config.AttributeMap, kit *Kit) (name string, config config.AttributeMap, err error) {
	names := etc.Keys()
	switch len(names) {
	case 0:
		err = fmt.Errorf("no peer list or chooser provided in config, need one of: %+v", kit.peerChooserAndListSpecNames())
	case 1:
		name = names[0]
		_, err = etc.Pop(name, &config)
	default:
		err = fmt.Errorf("unrecognized attributes in outbound config: %+v", etc)
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
func buildPeerListUpdater(c config.AttributeMap, identify func(string) peer.Identifier, kit *Kit) (peer.Binder, error) {
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
	case 1:
		// fall through to logic below
	default:
		sort.Strings(foundUpdaters) // deterministic error message
		return nil, fmt.Errorf(
			"found too many peer list updaters in config: got %s",
			strings.Join(foundUpdaters, ", "))
	}

	var peerListUpdaterConfig config.AttributeMap
	if _, err := c.Pop(foundUpdaters[0], &peerListUpdaterConfig); err != nil {
		return nil, err
	}

	// This decodes all attributes on the peer list updater block, including the
	// field with the name of the peer list updater.
	peerListUpdaterBuilder, err := peerListUpdaterSpec.PeerListUpdater.Decode(peerListUpdaterConfig, config.InterpolateWith(kit.resolver))
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

func configNames(c config.AttributeMap) (names []string) {
	for name := range c {
		names = append(names, name)
	}
	sort.Strings(names)
	return
}
