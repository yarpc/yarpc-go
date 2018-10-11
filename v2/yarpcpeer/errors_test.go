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

package yarpcpeer_test

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/v2/yarpcpeer"
	"go.uber.org/yarpc/v2/yarpctest"
)

func TestErrPeerHasNoReferenceToSubscriber(t *testing.T) {
	ctrl := gomock.NewController(t)
	identifier := yarpctest.NewMockIdentifier(ctrl)
	subscriber := yarpctest.NewMockSubscriber(ctrl)

	wantErr := fmt.Sprintf("peer (%v) has no reference to peer subscriber (%v)", identifier, subscriber)

	err := &yarpcpeer.ErrPeerHasNoReferenceToSubscriber{PeerIdentifier: identifier, PeerSubscriber: subscriber}
	assert.Equal(t, wantErr, err.Error())
}

func TestErrDialerHasNoReferenceToPeer2(t *testing.T) {
	dialerName := "test-dialer"
	peerIdentifier := "test-peer-id"

	wantErr := fmt.Sprintf("dialer %q has no reference to peer %q", dialerName, peerIdentifier)

	err := &yarpcpeer.ErrDialerHasNoReferenceToPeer{DialerName: dialerName, PeerIdentifier: peerIdentifier}
	assert.Equal(t, wantErr, err.Error())
}

func TestErrInvalidPeerType(t *testing.T) {
	expectedType := "test-type"
	peerIdentifier := yarpctest.NewMockIdentifier(gomock.NewController(t))

	wantErr := fmt.Sprintf("expected peer type (%s) but got peer (%v)", expectedType, peerIdentifier)

	err := &yarpcpeer.ErrInvalidPeerType{ExpectedType: expectedType, PeerIdentifier: peerIdentifier}
	assert.Equal(t, wantErr, err.Error())
}

func TestErrPeerListAlreadyStarted(t *testing.T) {
	peerList := "test-peer-list"
	wantErr := fmt.Sprintf("%s has already been started", peerList)

	err := yarpcpeer.ErrPeerListAlreadyStarted(peerList)
	assert.Equal(t, wantErr, err.Error())
}

func TestErrPeerListNotStarted(t *testing.T) {
	peerList := "test-peer-list"
	wantErr := fmt.Sprintf("%s has not been started or was stopped", peerList)

	err := yarpcpeer.ErrPeerListNotStarted(peerList)
	assert.Equal(t, wantErr, err.Error())
}

func TestErrInvalidPeerConversion(t *testing.T) {
	p := yarpctest.NewMockPeer(gomock.NewController(t))
	expectedType := "test-type"

	wantErr := fmt.Sprintf("cannot convert peer (%v) to type %s", p, expectedType)

	err := &yarpcpeer.ErrInvalidPeerConversion{Peer: p, ExpectedType: expectedType}
	assert.Equal(t, wantErr, err.Error())
}

func TestErrPeerAddAlreadyInList(t *testing.T) {
	p := "test-peer"
	wantErr := fmt.Sprintf("can't add peer %q because is already in peerlist", p)

	err := yarpcpeer.ErrPeerAddAlreadyInList(p)
	assert.Equal(t, wantErr, err.Error())
}

func TestErrPeerRemoveNotInList(t *testing.T) {
	p := "test-peer"
	wantErr := fmt.Sprintf("can't remove peer (%s) because it is not in peerlist", p)

	err := yarpcpeer.ErrPeerRemoveNotInList(p)
	assert.Equal(t, wantErr, err.Error())
}

func TestErrChooseContextHasNoDeadline(t *testing.T) {
	peerList := "test-peer"
	wantErr := fmt.Sprintf("can't wait for peer without a context deadline for peerlist %q", peerList)

	err := yarpcpeer.ErrChooseContextHasNoDeadline(peerList)
	assert.Equal(t, wantErr, err.Error())
}
