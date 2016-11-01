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
// Source: go.uber.org/yarpc/transport (interfaces: Handler,Registry)

package transporttest

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	transport "go.uber.org/yarpc/transport"
)

// Mock of Handler interface
type MockHandler struct {
	ctrl     *gomock.Controller
	recorder *_MockHandlerRecorder
}

// Recorder for MockHandler (not exported)
type _MockHandlerRecorder struct {
	mock *MockHandler
}

func NewMockHandler(ctrl *gomock.Controller) *MockHandler {
	mock := &MockHandler{ctrl: ctrl}
	mock.recorder = &_MockHandlerRecorder{mock}
	return mock
}

func (_m *MockHandler) EXPECT() *_MockHandlerRecorder {
	return _m.recorder
}

func (_m *MockHandler) Handle(_param0 context.Context, _param1 *transport.Request, _param2 transport.ResponseWriter) error {
	ret := _m.ctrl.Call(_m, "Handle", _param0, _param1, _param2)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockHandlerRecorder) Handle(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Handle", arg0, arg1, arg2)
}

// Mock of Registry interface
type MockRegistry struct {
	ctrl     *gomock.Controller
	recorder *_MockRegistryRecorder
}

// Recorder for MockRegistry (not exported)
type _MockRegistryRecorder struct {
	mock *MockRegistry
}

func NewMockRegistry(ctrl *gomock.Controller) *MockRegistry {
	mock := &MockRegistry{ctrl: ctrl}
	mock.recorder = &_MockRegistryRecorder{mock}
	return mock
}

func (_m *MockRegistry) EXPECT() *_MockRegistryRecorder {
	return _m.recorder
}

func (_m *MockRegistry) GetHandlerSpec(_param0 string, _param1 string) (transport.HandlerSpec, error) {
	ret := _m.ctrl.Call(_m, "GetHandlerSpec", _param0, _param1)
	ret0, _ := ret[0].(transport.HandlerSpec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockRegistryRecorder) GetHandlerSpec(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "GetHandlerSpec", arg0, arg1)
}

func (_m *MockRegistry) Register(_param0 []transport.Registrant) {
	_m.ctrl.Call(_m, "Register", _param0)
}

func (_mr *_MockRegistryRecorder) Register(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Register", arg0)
}

func (_m *MockRegistry) ServiceProcedures() []transport.ServiceProcedure {
	ret := _m.ctrl.Call(_m, "ServiceProcedures")
	ret0, _ := ret[0].([]transport.ServiceProcedure)
	return ret0
}

func (_mr *_MockRegistryRecorder) ServiceProcedures() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "ServiceProcedures")
}
