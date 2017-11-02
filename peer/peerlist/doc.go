// Package peerlist provides a utility for managing peer availability with a
// separate implementation of peer selection from just among available peers.
// The peer list implements the peer.ChooserList interface and accepts a
// peer.RetainedChooserList to provide the implementation-specific concern of,
// for example, a *roundrobin.List.
//
// The example is an implementation of peer.ChooserList using a random peer selection
// strategy, returned by newRandomRetainedPeerList(), implementing
// peer.RetainedChooserList.
//
//   type List struct {
//   	*peerlist.List
//   }
//
//   func New(transport peer.Transport) *List {
//   	return &List{
//   		List: peerlist.New(
//   			"random",
//   			transport,
//   			newRandomRetainedPeerList(),
//   		),
//   	}
//   }
//
package peerlist
