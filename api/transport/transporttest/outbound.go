// Code generated by MockGen. DO NOT EDIT.
// Source: go.uber.org/yarpc/api/transport (interfaces: UnaryOutbound,OnewayOutbound)

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

package transporttest

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	transport "go.uber.org/yarpc/api/transport"
	reflect "reflect"
)

// MockUnaryOutbound is a mock of UnaryOutbound interface
type MockUnaryOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockUnaryOutboundMockRecorder
}

// MockUnaryOutboundMockRecorder is the mock recorder for MockUnaryOutbound
type MockUnaryOutboundMockRecorder struct {
	mock *MockUnaryOutbound
}

// NewMockUnaryOutbound creates a new mock instance
func NewMockUnaryOutbound(ctrl *gomock.Controller) *MockUnaryOutbound {
	mock := &MockUnaryOutbound{ctrl: ctrl}
	mock.recorder = &MockUnaryOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (_m *MockUnaryOutbound) EXPECT() *MockUnaryOutboundMockRecorder {
	return _m.recorder
}

// Call mocks base method
func (_m *MockUnaryOutbound) Call(_param0 context.Context, _param1 *transport.Request) (*transport.Response, error) {
	ret := _m.ctrl.Call(_m, "Call", _param0, _param1)
	ret0, _ := ret[0].(*transport.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Call indicates an expected call of Call
func (_mr *MockUnaryOutboundMockRecorder) Call(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Call", reflect.TypeOf((*MockUnaryOutbound)(nil).Call), arg0, arg1)
}

// IsRunning mocks base method
func (_m *MockUnaryOutbound) IsRunning() bool {
	ret := _m.ctrl.Call(_m, "IsRunning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsRunning indicates an expected call of IsRunning
func (_mr *MockUnaryOutboundMockRecorder) IsRunning() *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "IsRunning", reflect.TypeOf((*MockUnaryOutbound)(nil).IsRunning))
}

// Start mocks base method
func (_m *MockUnaryOutbound) Start() error {
	ret := _m.ctrl.Call(_m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start
func (_mr *MockUnaryOutboundMockRecorder) Start() *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Start", reflect.TypeOf((*MockUnaryOutbound)(nil).Start))
}

// Stop mocks base method
func (_m *MockUnaryOutbound) Stop() error {
	ret := _m.ctrl.Call(_m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop
func (_mr *MockUnaryOutboundMockRecorder) Stop() *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Stop", reflect.TypeOf((*MockUnaryOutbound)(nil).Stop))
}

// Transports mocks base method
func (_m *MockUnaryOutbound) Transports() []transport.Transport {
	ret := _m.ctrl.Call(_m, "Transports")
	ret0, _ := ret[0].([]transport.Transport)
	return ret0
}

// Transports indicates an expected call of Transports
func (_mr *MockUnaryOutboundMockRecorder) Transports() *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Transports", reflect.TypeOf((*MockUnaryOutbound)(nil).Transports))
}

// MockOnewayOutbound is a mock of OnewayOutbound interface
type MockOnewayOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockOnewayOutboundMockRecorder
}

// MockOnewayOutboundMockRecorder is the mock recorder for MockOnewayOutbound
type MockOnewayOutboundMockRecorder struct {
	mock *MockOnewayOutbound
}

// NewMockOnewayOutbound creates a new mock instance
func NewMockOnewayOutbound(ctrl *gomock.Controller) *MockOnewayOutbound {
	mock := &MockOnewayOutbound{ctrl: ctrl}
	mock.recorder = &MockOnewayOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (_m *MockOnewayOutbound) EXPECT() *MockOnewayOutboundMockRecorder {
	return _m.recorder
}

// CallOneway mocks base method
func (_m *MockOnewayOutbound) CallOneway(_param0 context.Context, _param1 *transport.Request) (transport.Ack, error) {
	ret := _m.ctrl.Call(_m, "CallOneway", _param0, _param1)
	ret0, _ := ret[0].(transport.Ack)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CallOneway indicates an expected call of CallOneway
func (_mr *MockOnewayOutboundMockRecorder) CallOneway(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "CallOneway", reflect.TypeOf((*MockOnewayOutbound)(nil).CallOneway), arg0, arg1)
}

// IsRunning mocks base method
func (_m *MockOnewayOutbound) IsRunning() bool {
	ret := _m.ctrl.Call(_m, "IsRunning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsRunning indicates an expected call of IsRunning
func (_mr *MockOnewayOutboundMockRecorder) IsRunning() *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "IsRunning", reflect.TypeOf((*MockOnewayOutbound)(nil).IsRunning))
}

// Start mocks base method
func (_m *MockOnewayOutbound) Start() error {
	ret := _m.ctrl.Call(_m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start
func (_mr *MockOnewayOutboundMockRecorder) Start() *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Start", reflect.TypeOf((*MockOnewayOutbound)(nil).Start))
}

// Stop mocks base method
func (_m *MockOnewayOutbound) Stop() error {
	ret := _m.ctrl.Call(_m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop
func (_mr *MockOnewayOutboundMockRecorder) Stop() *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Stop", reflect.TypeOf((*MockOnewayOutbound)(nil).Stop))
}

// Transports mocks base method
func (_m *MockOnewayOutbound) Transports() []transport.Transport {
	ret := _m.ctrl.Call(_m, "Transports")
	ret0, _ := ret[0].([]transport.Transport)
	return ret0
}

// Transports indicates an expected call of Transports
func (_mr *MockOnewayOutboundMockRecorder) Transports() *gomock.Call {
	return _mr.mock.ctrl.RecordCallWithMethodType(_mr.mock, "Transports", reflect.TypeOf((*MockOnewayOutbound)(nil).Transports))
}
