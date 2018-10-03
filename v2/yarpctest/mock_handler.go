// Code generated by MockGen. DO NOT EDIT.
// Source: go.uber.org/yarpc/v2 (interfaces: UnaryTransportHandler,StreamTransportHandler)

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

// Package yarpctest is a generated GoMock package.
package yarpctest

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v2 "go.uber.org/yarpc/v2"
)

// MockUnaryHandler is a mock of UnaryTransportHandler interface
type MockUnaryHandler struct {
	ctrl     *gomock.Controller
	recorder *MockUnaryHandlerMockRecorder
}

// MockUnaryHandlerMockRecorder is the mock recorder for MockUnaryHandler
type MockUnaryHandlerMockRecorder struct {
	mock *MockUnaryHandler
}

// NewMockUnaryHandler creates a new mock instance
func NewMockUnaryHandler(ctrl *gomock.Controller) *MockUnaryHandler {
	mock := &MockUnaryHandler{ctrl: ctrl}
	mock.recorder = &MockUnaryHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockUnaryHandler) EXPECT() *MockUnaryHandlerMockRecorder {
	return m.recorder
}

// Handle mocks base method
func (m *MockUnaryHandler) Handle(arg0 context.Context, arg1 *v2.Request, arg2 *v2.Buffer) (*v2.Response, *v2.Buffer, error) {
	ret := m.ctrl.Call(m, "Handle", arg0, arg1, arg2)
	ret0, _ := ret[0].(*v2.Response)
	ret1, _ := ret[1].(*v2.Buffer)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Handle indicates an expected call of Handle
func (mr *MockUnaryHandlerMockRecorder) Handle(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Handle", reflect.TypeOf((*MockUnaryHandler)(nil).Handle), arg0, arg1, arg2)
}

// MockStreamHandler is a mock of StreamTransportHandler interface
type MockStreamHandler struct {
	ctrl     *gomock.Controller
	recorder *MockStreamHandlerMockRecorder
}

// MockStreamHandlerMockRecorder is the mock recorder for MockStreamHandler
type MockStreamHandlerMockRecorder struct {
	mock *MockStreamHandler
}

// NewMockStreamHandler creates a new mock instance
func NewMockStreamHandler(ctrl *gomock.Controller) *MockStreamHandler {
	mock := &MockStreamHandler{ctrl: ctrl}
	mock.recorder = &MockStreamHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockStreamHandler) EXPECT() *MockStreamHandlerMockRecorder {
	return m.recorder
}

// HandleStream mocks base method
func (m *MockStreamHandler) HandleStream(arg0 *v2.ServerStream) error {
	ret := m.ctrl.Call(m, "HandleStream", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// HandleStream indicates an expected call of HandleStream
func (mr *MockStreamHandlerMockRecorder) HandleStream(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleStream", reflect.TypeOf((*MockStreamHandler)(nil).HandleStream), arg0)
}
