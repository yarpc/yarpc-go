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

// TODO(mensch): This file is partially copied from v2/yarpctest/mock_stream.go.
// yarpctest currently fails to compile; remove this file when the package is
// ready for consumption.

package yarpcprotobuf

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v2 "go.uber.org/yarpc/v2"
)

// MockStreamCloser is a mock of StreamCloser interface
type MockStreamCloser struct {
	ctrl     *gomock.Controller
	recorder *MockStreamCloserMockRecorder
}

// MockStreamCloserMockRecorder is the mock recorder for MockStreamCloser
type MockStreamCloserMockRecorder struct {
	mock *MockStreamCloser
}

// NewMockStreamCloser creates a new mock instance
func NewMockStreamCloser(ctrl *gomock.Controller) *MockStreamCloser {
	mock := &MockStreamCloser{ctrl: ctrl}
	mock.recorder = &MockStreamCloserMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockStreamCloser) EXPECT() *MockStreamCloserMockRecorder {
	return m.recorder
}

// Close mocks base method
func (m *MockStreamCloser) Close(arg0 context.Context) error {
	ret := m.ctrl.Call(m, "Close", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockStreamCloserMockRecorder) Close(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockStreamCloser)(nil).Close), arg0)
}

// Context mocks base method
func (m *MockStreamCloser) Context() context.Context {
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context
func (mr *MockStreamCloserMockRecorder) Context() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockStreamCloser)(nil).Context))
}

// ReceiveMessage mocks base method
func (m *MockStreamCloser) ReceiveMessage(arg0 context.Context) (*v2.StreamMessage, error) {
	ret := m.ctrl.Call(m, "ReceiveMessage", arg0)
	ret0, _ := ret[0].(*v2.StreamMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReceiveMessage indicates an expected call of ReceiveMessage
func (mr *MockStreamCloserMockRecorder) ReceiveMessage(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReceiveMessage", reflect.TypeOf((*MockStreamCloser)(nil).ReceiveMessage), arg0)
}

// Request mocks base method
func (m *MockStreamCloser) Request() *v2.Request {
	ret := m.ctrl.Call(m, "Request")
	ret0, _ := ret[0].(*v2.Request)
	return ret0
}

// Request indicates an expected call of Request
func (mr *MockStreamCloserMockRecorder) Request() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Request", reflect.TypeOf((*MockStreamCloser)(nil).Request))
}

// SendMessage mocks base method
func (m *MockStreamCloser) SendMessage(arg0 context.Context, arg1 *v2.StreamMessage) error {
	ret := m.ctrl.Call(m, "SendMessage", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMessage indicates an expected call of SendMessage
func (mr *MockStreamCloserMockRecorder) SendMessage(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessage", reflect.TypeOf((*MockStreamCloser)(nil).SendMessage), arg0, arg1)
}
