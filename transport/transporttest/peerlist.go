// Automatically generated by MockGen. DO NOT EDIT!
// Source: go.uber.org/yarpc/transport (interfaces: PeerList)

package transporttest

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	transport "go.uber.org/yarpc/transport"
)

// Mock of PeerList interface
type MockPeerList struct {
	ctrl     *gomock.Controller
	recorder *_MockPeerListRecorder
}

// Recorder for MockPeerList (not exported)
type _MockPeerListRecorder struct {
	mock *MockPeerList
}

func NewMockPeerList(ctrl *gomock.Controller) *MockPeerList {
	mock := &MockPeerList{ctrl: ctrl}
	mock.recorder = &_MockPeerListRecorder{mock}
	return mock
}

func (_m *MockPeerList) EXPECT() *_MockPeerListRecorder {
	return _m.recorder
}

func (_m *MockPeerList) ChoosePeer(_param0 context.Context, _param1 *transport.Request) (transport.Peer, error) {
	ret := _m.ctrl.Call(_m, "ChoosePeer", _param0, _param1)
	ret0, _ := ret[0].(transport.Peer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockPeerListRecorder) ChoosePeer(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "ChoosePeer", arg0, arg1)
}

func (_m *MockPeerList) Start() error {
	ret := _m.ctrl.Call(_m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockPeerListRecorder) Start() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Start")
}

func (_m *MockPeerList) Stop() error {
	ret := _m.ctrl.Call(_m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockPeerListRecorder) Stop() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Stop")
}
