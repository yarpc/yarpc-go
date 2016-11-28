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
// Source: go.uber.org/yarpc/peer (interfaces: Chooser,ChangeListener)

package peertest

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	peer "go.uber.org/yarpc/peer"
	transport "go.uber.org/yarpc/transport"
)

// Mock of Chooser interface
type MockChooser struct {
	ctrl     *gomock.Controller
	recorder *_MockChooserRecorder
}

// Recorder for MockChooser (not exported)
type _MockChooserRecorder struct {
	mock *MockChooser
}

func NewMockChooser(ctrl *gomock.Controller) *MockChooser {
	mock := &MockChooser{ctrl: ctrl}
	mock.recorder = &_MockChooserRecorder{mock}
	return mock
}

func (_m *MockChooser) EXPECT() *_MockChooserRecorder {
	return _m.recorder
}

func (_m *MockChooser) ChoosePeer(_param0 context.Context, _param1 *transport.Request) (peer.Peer, error) {
	ret := _m.ctrl.Call(_m, "ChoosePeer", _param0, _param1)
	ret0, _ := ret[0].(peer.Peer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockChooserRecorder) ChoosePeer(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "ChoosePeer", arg0, arg1)
}

func (_m *MockChooser) Start() error {
	ret := _m.ctrl.Call(_m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockChooserRecorder) Start() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Start")
}

func (_m *MockChooser) Stop() error {
	ret := _m.ctrl.Call(_m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockChooserRecorder) Stop() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Stop")
}

// Mock of ChangeListener interface
type MockChangeListener struct {
	ctrl     *gomock.Controller
	recorder *_MockChangeListenerRecorder
}

// Recorder for MockChangeListener (not exported)
type _MockChangeListenerRecorder struct {
	mock *MockChangeListener
}

func NewMockChangeListener(ctrl *gomock.Controller) *MockChangeListener {
	mock := &MockChangeListener{ctrl: ctrl}
	mock.recorder = &_MockChangeListenerRecorder{mock}
	return mock
}

func (_m *MockChangeListener) EXPECT() *_MockChangeListenerRecorder {
	return _m.recorder
}

func (_m *MockChangeListener) Add(_param0 peer.Identifier) error {
	ret := _m.ctrl.Call(_m, "Add", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockChangeListenerRecorder) Add(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Add", arg0)
}

func (_m *MockChangeListener) Remove(_param0 peer.Identifier) error {
	ret := _m.ctrl.Call(_m, "Remove", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockChangeListenerRecorder) Remove(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Remove", arg0)
}
