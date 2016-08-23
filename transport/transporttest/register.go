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
// Source: github.com/yarpc/yarpc-go/transport (interfaces: Handler)

package transporttest

import (
	gomock "github.com/golang/mock/gomock"
	transport "github.com/yarpc/yarpc-go/transport"
	context "golang.org/x/net/context"
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

func (_m *MockHandler) Handle(_param0 context.Context, _param1 transport.Options, _param2 *transport.Request, _param3 transport.ResponseWriter) error {
	ret := _m.ctrl.Call(_m, "Handle", _param0, _param1, _param2, _param3)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockHandlerRecorder) Handle(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Handle", arg0, arg1, arg2, arg3)
}
