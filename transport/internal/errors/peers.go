package errors

import (
	"fmt"

	"go.uber.org/yarpc/transport"
)

// ErrPeerHasNoReferenceToSubscriber is called when a Peer is expected
// to operate on a PeerSubscriber it has no reference to
type ErrPeerHasNoReferenceToSubscriber struct {
	PeerIdentifier transport.PeerIdentifier
	PeerSubscriber transport.PeerSubscriber
}

func (e ErrPeerHasNoReferenceToSubscriber) Error() string {
	return fmt.Sprintf("peer (%v) has no reference to peer subscriber (%v)", e.PeerIdentifier, e.PeerSubscriber)
}

// ErrAgentHasNoReferenceToPeer is called when an agent is expected to
// operate on a Peer it has no reference to
type ErrAgentHasNoReferenceToPeer struct {
	Agent          transport.PeerAgent
	PeerIdentifier transport.PeerIdentifier
}

func (e ErrAgentHasNoReferenceToPeer) Error() string {
	return fmt.Sprintf("agent (%v) has no reference to peer (%v)", e.Agent, e.PeerIdentifier)
}

// ErrInvalidPeerType is when a specfic peer type is required, but
// was not passed in
type ErrInvalidPeerType struct {
	ExpectedType   string
	PeerIdentifier transport.PeerIdentifier
}

func (e ErrInvalidPeerType) Error() string {
	return fmt.Sprintf("expected peer type (%s) but got peer (%v)", e.ExpectedType, e.PeerIdentifier)
}

// ErrPeerListAlreadyStarted represents a failure because Start() was already
// called on the peerlist.
type ErrPeerListAlreadyStarted string

func (e ErrPeerListAlreadyStarted) Error() string {
	return fmt.Sprintf("%s has already been started", string(e))
}

// ErrPeerListNotStarted represents a failure because Start() was not called
// on a peerlist or if Stop() was called.
type ErrPeerListNotStarted string

func (e ErrPeerListNotStarted) Error() string {
	return fmt.Sprintf("%s has not been started or was stopped", string(e))
}

// ErrInvalidPeerConversion is called when a peer can't be properly converted
type ErrInvalidPeerConversion struct {
	Peer         transport.Peer
	ExpectedType string
}

func (e ErrInvalidPeerConversion) Error() string {
	return fmt.Sprintf("cannot convert peer (%v) to type %s", e.Peer, e.ExpectedType)
}

// ErrInvalidAgentConversion is called when an agent can't be properly converted
type ErrInvalidAgentConversion struct {
	Agent        transport.PeerAgent
	ExpectedType string
}

func (e ErrInvalidAgentConversion) Error() string {
	return fmt.Sprintf("cannot convert agent (%v) to type %s", e.Agent, e.ExpectedType)
}
