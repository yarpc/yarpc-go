// Code generated by MockGen. DO NOT EDIT.
// Source: go.uber.org/yarpc/api/peer (interfaces: Chooser,List,ChooserList)

// Package peertest is a generated GoMock package.
package peertest

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	peer "go.uber.org/yarpc/api/peer"
	transport "go.uber.org/yarpc/api/transport"
	reflect "reflect"
)

// MockChooser is a mock of Chooser interface
type MockChooser struct {
	ctrl     *gomock.Controller
	recorder *MockChooserMockRecorder
}

// MockChooserMockRecorder is the mock recorder for MockChooser
type MockChooserMockRecorder struct {
	mock *MockChooser
}

// NewMockChooser creates a new mock instance
func NewMockChooser(ctrl *gomock.Controller) *MockChooser {
	mock := &MockChooser{ctrl: ctrl}
	mock.recorder = &MockChooserMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockChooser) EXPECT() *MockChooserMockRecorder {
	return m.recorder
}

// Choose mocks base method
func (m *MockChooser) Choose(arg0 context.Context, arg1 *transport.Request) (peer.Peer, func(error), error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Choose", arg0, arg1)
	ret0, _ := ret[0].(peer.Peer)
	ret1, _ := ret[1].(func(error))
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Choose indicates an expected call of Choose
func (mr *MockChooserMockRecorder) Choose(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Choose", reflect.TypeOf((*MockChooser)(nil).Choose), arg0, arg1)
}

// IsRunning mocks base method
func (m *MockChooser) IsRunning() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsRunning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsRunning indicates an expected call of IsRunning
func (mr *MockChooserMockRecorder) IsRunning() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsRunning", reflect.TypeOf((*MockChooser)(nil).IsRunning))
}

// Start mocks base method
func (m *MockChooser) Start() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start
func (mr *MockChooserMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockChooser)(nil).Start))
}

// Stop mocks base method
func (m *MockChooser) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop
func (mr *MockChooserMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockChooser)(nil).Stop))
}

// MockList is a mock of List interface
type MockList struct {
	ctrl     *gomock.Controller
	recorder *MockListMockRecorder
}

// MockListMockRecorder is the mock recorder for MockList
type MockListMockRecorder struct {
	mock *MockList
}

// NewMockList creates a new mock instance
func NewMockList(ctrl *gomock.Controller) *MockList {
	mock := &MockList{ctrl: ctrl}
	mock.recorder = &MockListMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockList) EXPECT() *MockListMockRecorder {
	return m.recorder
}

// Update mocks base method
func (m *MockList) Update(arg0 peer.ListUpdates) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update
func (mr *MockListMockRecorder) Update(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockList)(nil).Update), arg0)
}

// MockChooserList is a mock of ChooserList interface
type MockChooserList struct {
	ctrl     *gomock.Controller
	recorder *MockChooserListMockRecorder
}

// MockChooserListMockRecorder is the mock recorder for MockChooserList
type MockChooserListMockRecorder struct {
	mock *MockChooserList
}

// NewMockChooserList creates a new mock instance
func NewMockChooserList(ctrl *gomock.Controller) *MockChooserList {
	mock := &MockChooserList{ctrl: ctrl}
	mock.recorder = &MockChooserListMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockChooserList) EXPECT() *MockChooserListMockRecorder {
	return m.recorder
}

// Choose mocks base method
func (m *MockChooserList) Choose(arg0 context.Context, arg1 *transport.Request) (peer.Peer, func(error), error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Choose", arg0, arg1)
	ret0, _ := ret[0].(peer.Peer)
	ret1, _ := ret[1].(func(error))
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Choose indicates an expected call of Choose
func (mr *MockChooserListMockRecorder) Choose(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Choose", reflect.TypeOf((*MockChooserList)(nil).Choose), arg0, arg1)
}

// IsRunning mocks base method
func (m *MockChooserList) IsRunning() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsRunning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsRunning indicates an expected call of IsRunning
func (mr *MockChooserListMockRecorder) IsRunning() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsRunning", reflect.TypeOf((*MockChooserList)(nil).IsRunning))
}

// Start mocks base method
func (m *MockChooserList) Start() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start
func (mr *MockChooserListMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockChooserList)(nil).Start))
}

// Stop mocks base method
func (m *MockChooserList) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop
func (mr *MockChooserListMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockChooserList)(nil).Stop))
}

// Update mocks base method
func (m *MockChooserList) Update(arg0 peer.ListUpdates) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update
func (mr *MockChooserListMockRecorder) Update(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockChooserList)(nil).Update), arg0)
}
