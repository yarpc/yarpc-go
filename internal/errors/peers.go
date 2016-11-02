package errors

import "fmt"

// ErrAgentHasNoReferenceToPeer is called when an agent is expected to operate on a Peer it has no reference to
type ErrAgentHasNoReferenceToPeer struct {
	Agent          interface{}
	PeerIdentifier interface{}
}

func (e ErrAgentHasNoReferenceToPeer) Error() string {
	return fmt.Sprintf("agent (%v) has no reference to peer (%v)", e.Agent, e.PeerIdentifier)
}
