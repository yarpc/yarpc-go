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
	// TODO all private
	with   string   `config:"with"`   // like peers / file / dns / dns-srv
	choose string   `config:"choose"` // like round-robin / least-pending
	peer   string   `config:"peer"`   // implies with: peer
	peers  []string `config:"peers"`  // implies with: peers
	etc    attributeMap
}

// Decode captures a configuration on this ChooserConfig struct.
func (c ChooserConfig) Decode(into decode.Into) error {
	var err error

	err = into(&c.etc)
	if err != nil {
		return err
	}

	c.with, err = c.etc.PopString("with")
	if err != nil {
		// TODO error
		return err
	}

	c.choose, err = c.etc.PopString("choose")
	if err != nil {
		// TODO error
		return err
	}

	c.peer, err = c.etc.PopString("peer")
	if err != nil {
		// TODO error
		return err
	}

	_, err = c.etc.Pop("peers", &c.peers)
	if err != nil {
		// TODO error
		return err
	}

	return nil
}

// BuildChooser translates a chooser configuration into a peer chooser, backed
// by a peer list bound to a peer list binder.
func (c ChooserConfig) BuildChooser(transport peer.Transport, identify func(string) peer.Identifier, kit *Kit) (peer.Chooser, error) {
	if c.peer != "" {
		// TODO validate, no further config
		return peerbind.NewSingle(identify(c.peer), transport), nil
	}

	// Build a peer list for a peer selection strategy, like round-robin or
	// least-pending.
	// Determined by:
	//  choose: ?
	list, err := c.BuildList(transport, kit)
	if err != nil {
		return nil, err
	}

	// Build a peer list binder, using "peer", "peers", or a custom peer
	// binder.
	binder, err := c.BuildBinder(transport, identify, kit)
	if err != nil {
		return nil, err
	}

	return peerbind.Bind(list, binder), nil
}

// BuildBinder translates a chooser configuration to a peer list binder. The
// binder is suitable for getting updates for the contents of a peer list, but
// not for selecting peers.
func (c ChooserConfig) BuildBinder(transport peer.Transport, identify func(string) peer.Identifier, kit *Kit) (peerbind.Binder, error) {
	// establish peers to bind
	if c.peers != nil {
		// TODO does not go well with "with:"
		return peerbind.BindPeers(identifyAll(identify, c.peers)), nil
	}

	binderSpec := kit.binder(c.with)
	if binderSpec == nil {
		return nil, fmt.Errorf("not a support peer list binder %q", c.with)
	}

	binderBuilder, err := binderSpec.Binder.Decode(c.etc)
	if err != nil {
		return nil, err // TODO wrap
	}

	result, err := binderBuilder.Build(kit)
	if err != nil {
		return nil, err // TODO wrap
	}

	return result.(peerbind.Binder), nil
}

// BuildList translates a peer chooser configuration to a peer list / peer
// chooser, suitable for a peer selection strategy.  The list will be empty
// until updated with a peer list binder.
func (c ChooserConfig) BuildList(transport peer.Transport, kit *Kit) (peer.ChooserList, error) {
	switch c.choose {
	case "":
		fallthrough
	case "least-pending":
		return peerheap.New(transport), nil
	case "round-robin":
		return roundrobin.New(transport), nil
	}

	chooserSpec := kit.chooser(c.choose)
	if chooserSpec == nil {
		// TODO thread more context for errors on the location of the peer chooser
		return nil, fmt.Errorf("not a supported peer list chooser %q", c.choose)
	}

	chooserBuilder, err := chooserSpec.Chooser.Decode(c.etc)
	if err != nil {
		return nil, err // TODO wrap
	}

	result, err := chooserBuilder.Build(kit)
	if err != nil {
		return nil, err // TODO wrap
	}

	return result.(peer.ChooserList), nil
}

func identifyAll(identify func(string) peer.Identifier, peers []string) []peer.Identifier {
	pids := make([]peer.Identifier, len(peers))
	for i, peer := range peers {
		pids[i] = identify(peer)
	}
	return pids
}
