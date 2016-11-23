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

// Automatically generated by MockGen. DO NOT EDIT!
// Source: go.uber.org/yarpc/peer (interfaces: Identifier,Peer)

package peertest

import (
	gomock "github.com/golang/mock/gomock"
	peer "go.uber.org/yarpc/peer"
)

// Mock of Identifier interface
type MockIdentifier struct {
	ctrl     *gomock.Controller
	recorder *_MockIdentifierRecorder
}

// Recorder for MockIdentifier (not exported)
type _MockIdentifierRecorder struct {
	mock *MockIdentifier
}

func NewMockIdentifier(ctrl *gomock.Controller) *MockIdentifier {
	mock := &MockIdentifier{ctrl: ctrl}
	mock.recorder = &_MockIdentifierRecorder{mock}
	return mock
}

func (_m *MockIdentifier) EXPECT() *_MockIdentifierRecorder {
	return _m.recorder
}

func (_m *MockIdentifier) Identifier() string {
	ret := _m.ctrl.Call(_m, "Identifier")
	ret0, _ := ret[0].(string)
	return ret0
}

func (_mr *_MockIdentifierRecorder) Identifier() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Identifier")
}

// Mock of Peer interface
type MockPeer struct {
	ctrl     *gomock.Controller
	recorder *_MockPeerRecorder
}

// Recorder for MockPeer (not exported)
type _MockPeerRecorder struct {
	mock *MockPeer
}

func NewMockPeer(ctrl *gomock.Controller) *MockPeer {
	mock := &MockPeer{ctrl: ctrl}
	mock.recorder = &_MockPeerRecorder{mock}
	return mock
}

func (_m *MockPeer) EXPECT() *_MockPeerRecorder {
	return _m.recorder
}

func (_m *MockPeer) EndRequest() {
	_m.ctrl.Call(_m, "EndRequest")
}

func (_mr *_MockPeerRecorder) EndRequest() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "EndRequest")
}

func (_m *MockPeer) Identifier() string {
	ret := _m.ctrl.Call(_m, "Identifier")
	ret0, _ := ret[0].(string)
	return ret0
}

func (_mr *_MockPeerRecorder) Identifier() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Identifier")
}

func (_m *MockPeer) StartRequest() {
	_m.ctrl.Call(_m, "StartRequest")
}

func (_mr *_MockPeerRecorder) StartRequest() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "StartRequest")
}

func (_m *MockPeer) Status() peer.Status {
	ret := _m.ctrl.Call(_m, "Status")
	ret0, _ := ret[0].(peer.Status)
	return ret0
}

func (_mr *_MockPeerRecorder) Status() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Status")
}
