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
// Source: go.uber.org/yarpc/api/transport (interfaces: Inbound)

package transporttest

import (
	gomock "github.com/golang/mock/gomock"
	transport "go.uber.org/yarpc/api/transport"
)

// Mock of Inbound interface
type MockInbound struct {
	ctrl     *gomock.Controller
	recorder *_MockInboundRecorder
}

// Recorder for MockInbound (not exported)
type _MockInboundRecorder struct {
	mock *MockInbound
}

func NewMockInbound(ctrl *gomock.Controller) *MockInbound {
	mock := &MockInbound{ctrl: ctrl}
	mock.recorder = &_MockInboundRecorder{mock}
	return mock
}

func (_m *MockInbound) EXPECT() *_MockInboundRecorder {
	return _m.recorder
}

func (_m *MockInbound) IsRunning() bool {
	ret := _m.ctrl.Call(_m, "IsRunning")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (_mr *_MockInboundRecorder) IsRunning() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "IsRunning")
}

func (_m *MockInbound) SetRouter(_param0 transport.Router) {
	_m.ctrl.Call(_m, "SetRouter", _param0)
}

func (_mr *_MockInboundRecorder) SetRouter(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SetRouter", arg0)
}

func (_m *MockInbound) Start() error {
	ret := _m.ctrl.Call(_m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockInboundRecorder) Start() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Start")
}

func (_m *MockInbound) Stop() error {
	ret := _m.ctrl.Call(_m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockInboundRecorder) Stop() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Stop")
}

func (_m *MockInbound) Transports() []transport.Transport {
	ret := _m.ctrl.Call(_m, "Transports")
	ret0, _ := ret[0].([]transport.Transport)
	return ret0
}

func (_mr *_MockInboundRecorder) Transports() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Transports")
}
