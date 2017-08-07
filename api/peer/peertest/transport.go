// Code generated by MockGen. DO NOT EDIT.
// Source: go.uber.org/yarpc/api/peer (interfaces: Transport,Subscriber)

// Copyright (c) 2017 Uber Technologies, Inc.
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

package peertest

import (
	gomock "github.com/golang/mock/gomock"
	peer "go.uber.org/yarpc/api/peer"
	reflect "reflect"
)

// MockTransport is a mock of Transport interface
type MockTransport struct {
	ctrl     *gomock.Controller
	recorder *MockTransportMockRecorder
}

// MockTransportMockRecorder is the mock recorder for MockTransport
type MockTransportMockRecorder struct {
	mock *MockTransport
}

// NewMockTransport creates a new mock instance
func NewMockTransport(ctrl *gomock.Controller) *MockTransport {
	mock := &MockTransport{ctrl: ctrl}
	mock.recorder = &MockTransportMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (_m *MockTransport) EXPECT() *MockTransportMockRecorder {
	return _m.recorder
}

// ReleasePeer mocks base method
func (_m *MockTransport) ReleasePeer(_param0 peer.Identifier, _param1 peer.Subscriber) error {
	ret := _m.ctrl.Call(_m, "ReleasePeer", _param0, _param1)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReleasePeer indicates an expected call of ReleasePeer
func (_mr *MockTransportMockRecorder) ReleasePeer(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "ReleasePeer", reflect.TypeOf((*MockTransport)(nil).ReleasePeer), arg0, arg1)
}

// RetainPeer mocks base method
func (_m *MockTransport) RetainPeer(_param0 peer.Identifier, _param1 peer.Subscriber) (peer.Peer, error) {
	ret := _m.ctrl.Call(_m, "RetainPeer", _param0, _param1)
	ret0, _ := ret[0].(peer.Peer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RetainPeer indicates an expected call of RetainPeer
func (_mr *MockTransportMockRecorder) RetainPeer(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "RetainPeer", reflect.TypeOf((*MockTransport)(nil).RetainPeer), arg0, arg1)
}

// MockSubscriber is a mock of Subscriber interface
type MockSubscriber struct {
	ctrl     *gomock.Controller
	recorder *MockSubscriberMockRecorder
}

// MockSubscriberMockRecorder is the mock recorder for MockSubscriber
type MockSubscriberMockRecorder struct {
	mock *MockSubscriber
}

// NewMockSubscriber creates a new mock instance
func NewMockSubscriber(ctrl *gomock.Controller) *MockSubscriber {
	mock := &MockSubscriber{ctrl: ctrl}
	mock.recorder = &MockSubscriberMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (_m *MockSubscriber) EXPECT() *MockSubscriberMockRecorder {
	return _m.recorder
}

// NotifyStatusChanged mocks base method
func (_m *MockSubscriber) NotifyStatusChanged(_param0 peer.Identifier) {
	_m.ctrl.Call(_m, "NotifyStatusChanged", _param0)
}

// NotifyStatusChanged indicates an expected call of NotifyStatusChanged
func (_mr *MockSubscriberMockRecorder) NotifyStatusChanged(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "NotifyStatusChanged", reflect.TypeOf((*MockSubscriber)(nil).NotifyStatusChanged), arg0)
}
