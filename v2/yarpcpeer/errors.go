// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpcpeer

import (
	"fmt"

	yarpc "go.uber.org/yarpc/v2"
)

// ErrMissingDialer indicates that direct calls to an address are not possible
// with a particular outbound configuration since the outbound was created
// without a Dialer, probably with a Chooser instead.
type ErrMissingDialer struct {
	Transport string
}

func (e ErrMissingDialer) Error() string {
	return fmt.Sprintf("can't call direct address on peer on %q outbound that does not have a dialer", e.Transport)
}

// ErrPeerHasNoReferenceToSubscriber is called when a Peer is expected
// to operate on a PeerSubscriber it has no reference to
type ErrPeerHasNoReferenceToSubscriber struct {
	PeerIdentifier yarpc.Identifier
	PeerSubscriber yarpc.Subscriber
}

func (e ErrPeerHasNoReferenceToSubscriber) Error() string {
	return fmt.Sprintf("peer (%v) has no reference to peer subscriber (%v)", e.PeerIdentifier, e.PeerSubscriber)
}

// ErrDialerHasNoReferenceToPeer is called when a transport is expected to
// operate on a Peer it has no reference to
type ErrDialerHasNoReferenceToPeer struct {
	DialerName     string
	PeerIdentifier string
}

func (e ErrDialerHasNoReferenceToPeer) Error() string {
	return fmt.Sprintf("dialer %q has no reference to peer %q", e.DialerName, e.PeerIdentifier)
}

// ErrInvalidPeerType is when a specfic peer type is required, but
// was not passed in
type ErrInvalidPeerType struct {
	ExpectedType   string
	PeerIdentifier yarpc.Identifier
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
	Peer         yarpc.Peer
	ExpectedType string
}

func (e ErrInvalidPeerConversion) Error() string {
	return fmt.Sprintf("cannot convert peer (%v) to type %s", e.Peer, e.ExpectedType)
}

// ErrPeerAddAlreadyInList is returned to peer list updater if the
// peerlist is already tracking a peer for the added identifier
type ErrPeerAddAlreadyInList string

func (e ErrPeerAddAlreadyInList) Error() string {
	return fmt.Sprintf("can't add peer %q because is already in peerlist", string(e))
}

// ErrPeerRemoveNotInList is returned to peer list updater if the peerlist
// is not tracking the peer to remove for a given identifier
type ErrPeerRemoveNotInList string

func (e ErrPeerRemoveNotInList) Error() string {
	return fmt.Sprintf("can't remove peer (%s) because it is not in peerlist", string(e))
}

// ErrChooseContextHasNoDeadline is returned when a context is sent to a peerlist with no deadline
// DEPRECATED use yarpcerror api instead.
type ErrChooseContextHasNoDeadline string

func (e ErrChooseContextHasNoDeadline) Error() string {
	return fmt.Sprintf("can't wait for peer without a context deadline for peerlist %q", string(e))
}
