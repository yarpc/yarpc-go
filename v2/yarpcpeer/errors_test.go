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

func TestErrTransportHasNoReferenceToPeer2(t *testing.T) {
	transportName := "test-transport"
	peerIdentifier := "test-peer-id"

	wantErr := fmt.Sprintf("transport %q has no reference to peer %q", transportName, peerIdentifier)

	err := &yarpcpeer.ErrTransportHasNoReferenceToPeer{TransportName: transportName, PeerIdentifier: peerIdentifier}
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
