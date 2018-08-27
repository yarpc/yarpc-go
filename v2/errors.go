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

package yarpc

import (
	"fmt"

	"go.uber.org/yarpc/v2/yarpcerrors"
)

// InboundBadRequestError builds an error which indicates that an inbound
// cannot process a request because it is a bad request.
//
// IsBadRequestError returns true for these errors.
//
// Deprecated: use yarpcerrors.Newf with yarpcerrors.CodeInvalidArgument instead.
func InboundBadRequestError(err error) error {
	return yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, err.Error())
}

// IsBadRequestError returns true if the request could not be processed
// because it was invalid.
//
// Deprecated: use yarpcerrors.FromError(err).Code() == yarpcerrors.CodeInvalidArgument instead.
func IsBadRequestError(err error) bool {
	return yarpcerrors.FromError(err).Code() == yarpcerrors.CodeInvalidArgument
}

// IsUnexpectedError returns true if the server panicked or failed to process
// the request with an unhandled error.
//
// Deprecated: use yarpcerrors.FromError(err).Code() == yarpcerrors.CodeInternal instead.
func IsUnexpectedError(err error) bool {
	return yarpcerrors.FromError(err).Code() == yarpcerrors.CodeInternal
}

// IsTimeoutError return true if the given error is a TimeoutError.
//
// Deprecated: use yarpcerrors.FromError(err).Code() == yarpcerrors.CodeDeadlineExceeded instead.
func IsTimeoutError(err error) bool {
	return yarpcerrors.FromError(err).Code() == yarpcerrors.CodeDeadlineExceeded
}

// UnrecognizedProcedureError returns an error for the given request,
// such that IsUnrecognizedProcedureError can distinguish it from other errors
// coming out of router.Choose.
//
// Deprecated: use yarpcerrors.Newf with yarpcerrors.CodeUnimplemented instead.
func UnrecognizedProcedureError(req *Request) error {
	return yarpcerrors.Newf(yarpcerrors.CodeUnimplemented, "unrecognized procedure %q for service %q", req.Procedure, req.Service)
}

// IsUnrecognizedProcedureError returns true for errors returned by
// Router.Choose if the router cannot find a handler for the request.
//
// Deprecated: use yarpcerrors.FromError(err).Code() == yarpcerrors.CodeUnimplemented instead.
func IsUnrecognizedProcedureError(err error) bool {
	return yarpcerrors.FromError(err).Code() == yarpcerrors.CodeUnimplemented
}

// ErrPeerHasNoReferenceToSubscriber is called when a Peer is expected
// to operate on a PeerSubscriber it has no reference to
type ErrPeerHasNoReferenceToSubscriber struct {
	PeerIdentifier Identifier
	PeerSubscriber Subscriber
}

func (e ErrPeerHasNoReferenceToSubscriber) Error() string {
	return fmt.Sprintf("peer (%v) has no reference to peer subscriber (%v)", e.PeerIdentifier, e.PeerSubscriber)
}

// ErrTransportHasNoReferenceToPeer is called when a transport is expected to
// operate on a Peer it has no reference to
type ErrTransportHasNoReferenceToPeer struct {
	TransportName  string
	PeerIdentifier string
}

func (e ErrTransportHasNoReferenceToPeer) Error() string {
	return fmt.Sprintf("transport %q has no reference to peer %q", e.TransportName, e.PeerIdentifier)
}

// ErrInvalidPeerType is when a specfic peer type is required, but
// was not passed in
type ErrInvalidPeerType struct {
	ExpectedType   string
	PeerIdentifier Identifier
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
	Peer         Peer
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
// DEPRECATED use yarpcerrors api instead.
type ErrChooseContextHasNoDeadline string

func (e ErrChooseContextHasNoDeadline) Error() string {
	return fmt.Sprintf("can't wait for peer without a context deadline for peerlist %q", string(e))
}
