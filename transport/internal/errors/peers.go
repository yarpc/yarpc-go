// Copyright (c) 2016 Uber Technologies, Inc.
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
	Agent          transport.Agent
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
	Agent        transport.Agent
	ExpectedType string
}

func (e ErrInvalidAgentConversion) Error() string {
	return fmt.Sprintf("cannot convert agent (%v) to type %s", e.Agent, e.ExpectedType)
}

// ErrNoPeerToSelect is used when a peerlist doesn't have any peers to return
type ErrNoPeerToSelect string

func (e ErrNoPeerToSelect) Error() string {
	return fmt.Sprintf("could not find a peer to select in peerlist %s", string(e))
}

// ErrPeerAlreadyInList is used when a peerlist attempts to add a peer that is already in the list
type ErrPeerAlreadyInList struct {
	Peer transport.Peer
}

func (e ErrPeerAlreadyInList) Error() string {
	return fmt.Sprintf("can't add peer (%v) because is already in peerlist", e.Peer)
}

// ErrPeerNotInList is used when a peerlist attempts to remove a peer that is not in the list
type ErrPeerNotInList struct {
	PeerIdentifier transport.PeerIdentifier
}

func (e ErrPeerNotInList) Error() string {
	return fmt.Sprintf("can't remove peer (%v) because it is not in peerlist", e.PeerIdentifier)
}

// Errors is used to encapsulate multiple errors returned at the same time
type Errors []error

func (e Errors) Error() string {
	if len(e) == 1 {
		return e[0].Error()
	}

	msg := "multiple peer errors:"
	for _, err := range e {
		msg += "\n" + err.Error()
	}
	return msg
}
