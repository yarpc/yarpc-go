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

package yarpc_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	. "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerrors"
	"go.uber.org/yarpc/v2/yarpctest"
)

func TestBadRequestError(t *testing.T) {
	err := errors.New("derp")
	err = InboundBadRequestError(err)
	assert.True(t, IsBadRequestError(err))
}

func TestIsUnexpectedError(t *testing.T) {
	assert.True(t, IsUnexpectedError(yarpcerrors.Newf(yarpcerrors.CodeInternal, "")))
}

func TestIsTimeoutError(t *testing.T) {
	assert.True(t, IsTimeoutError(yarpcerrors.Newf(yarpcerrors.CodeDeadlineExceeded, "")))
}

func TestUnrecognizedProcedureError(t *testing.T) {
	err := UnrecognizedProcedureError(&Request{Service: "curly", Procedure: "nyuck"})
	assert.True(t, IsUnrecognizedProcedureError(err))
	assert.False(t, IsUnrecognizedProcedureError(errors.New("derp")))
}

func TestExpectEncodings(t *testing.T) {
	assert.Error(t, ExpectEncodings(&Request{}, "foo"))
	assert.NoError(t, ExpectEncodings(&Request{Encoding: "foo"}, "foo"))
	assert.NoError(t, ExpectEncodings(&Request{Encoding: "foo"}, "foo", "bar"))
	assert.Error(t, ExpectEncodings(&Request{Encoding: "foo"}, "bar"))
	assert.Error(t, ExpectEncodings(&Request{Encoding: "foo"}, "bar", "baz"))
}

func TestEncodeErrors(t *testing.T) {
	tests := []struct {
		errorFunc     func(*Request, error) error
		expectedCode  yarpcerrors.Code
		expectedWords []string
	}{
		{
			errorFunc:     RequestBodyEncodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"request", "body", "encode"},
		},
		{
			errorFunc:     RequestHeadersEncodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"request", "headers", "encode"},
		},
		{
			errorFunc:     RequestBodyDecodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"request", "body", "decode"},
		},
		{
			errorFunc:     RequestHeadersDecodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"request", "headers", "decode"},
		},
		{
			errorFunc:     ResponseBodyEncodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"response", "body", "encode"},
		},
		{
			errorFunc:     ResponseHeadersEncodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"response", "headers", "encode"},
		},
		{
			errorFunc:     ResponseBodyDecodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"response", "body", "decode"},
		},
		{
			errorFunc:     ResponseHeadersDecodeError,
			expectedCode:  yarpcerrors.CodeInvalidArgument,
			expectedWords: []string{"response", "headers", "decode"},
		},
	}
	request := &Request{}
	for _, tt := range tests {
		assertError(t, tt.errorFunc(request, errors.New("")), tt.expectedCode, tt.expectedWords...)
	}
}

func assertError(t *testing.T, err error, expectedCode yarpcerrors.Code, expectedWords ...string) {
	assert.Error(t, err)
	assert.Equal(t, expectedCode, yarpcerrors.FromError(err).Code())
	for _, expectedWord := range expectedWords {
		assert.Contains(t, err.Error(), expectedWord)
	}
}

func TestErrPeerHasNoReferenceToSubscriber(t *testing.T) {
	ctrl := gomock.NewController(t)
	identifier := yarpctest.NewMockIdentifier(ctrl)
	subscriber := yarpctest.NewMockSubscriber(ctrl)

	wantErr := fmt.Sprintf("peer (%v) has no reference to peer subscriber (%v)", identifier, subscriber)

	err := &ErrPeerHasNoReferenceToSubscriber{PeerIdentifier: identifier, PeerSubscriber: subscriber}
	assert.Equal(t, wantErr, err.Error())
}

func TestErrTransportHasNoReferenceToPeer2(t *testing.T) {
	transportName := "test-transport"
	peerIdentifier := "test-peer-id"

	wantErr := fmt.Sprintf("transport %q has no reference to peer %q", transportName, peerIdentifier)

	err := &ErrTransportHasNoReferenceToPeer{TransportName: transportName, PeerIdentifier: peerIdentifier}
	assert.Equal(t, wantErr, err.Error())
}

func TestErrInvalidPeerType(t *testing.T) {
	expectedType := "test-type"
	peerIdentifier := yarpctest.NewMockIdentifier(gomock.NewController(t))

	wantErr := fmt.Sprintf("expected peer type (%s) but got peer (%v)", expectedType, peerIdentifier)

	err := &ErrInvalidPeerType{ExpectedType: expectedType, PeerIdentifier: peerIdentifier}
	assert.Equal(t, wantErr, err.Error())
}

func TestErrPeerListAlreadyStarted(t *testing.T) {
	peerList := "test-peer-list"
	wantErr := fmt.Sprintf("%s has already been started", peerList)

	err := ErrPeerListAlreadyStarted(peerList)
	assert.Equal(t, wantErr, err.Error())
}

func TestErrPeerListNotStarted(t *testing.T) {
	peerList := "test-peer-list"
	wantErr := fmt.Sprintf("%s has not been started or was stopped", peerList)

	err := ErrPeerListNotStarted(peerList)
	assert.Equal(t, wantErr, err.Error())
}

func TestErrInvalidPeerConversion(t *testing.T) {
	p := yarpctest.NewMockPeer(gomock.NewController(t))
	expectedType := "test-type"

	wantErr := fmt.Sprintf("cannot convert peer (%v) to type %s", p, expectedType)

	err := &ErrInvalidPeerConversion{Peer: p, ExpectedType: expectedType}
	assert.Equal(t, wantErr, err.Error())
}

func TestErrPeerAddAlreadyInList(t *testing.T) {
	p := "test-peer"
	wantErr := fmt.Sprintf("can't add peer %q because is already in peerlist", p)

	err := ErrPeerAddAlreadyInList(p)
	assert.Equal(t, wantErr, err.Error())
}

func TestErrPeerRemoveNotInList(t *testing.T) {
	p := "test-peer"
	wantErr := fmt.Sprintf("can't remove peer (%s) because it is not in peerlist", p)

	err := ErrPeerRemoveNotInList(p)
	assert.Equal(t, wantErr, err.Error())
}

func TestErrChooseContextHasNoDeadline(t *testing.T) {
	peerList := "test-peer"
	wantErr := fmt.Sprintf("can't wait for peer without a context deadline for peerlist %q", peerList)

	err := ErrChooseContextHasNoDeadline(peerList)
	assert.Equal(t, wantErr, err.Error())
}
