// Automatically generated by MockGen. DO NOT EDIT!
// Source: go.uber.org/yarpc/encoding/thrift (interfaces: UnaryHandler)

package thrift

import (
	context "context"

	gomock "github.com/golang/mock/gomock"
	wire "go.uber.org/thriftrw/wire"
	yarpc "go.uber.org/yarpc"
)

// Mock of UnaryHandler interface
type MockUnaryHandler struct {
	ctrl     *gomock.Controller
	recorder *_MockUnaryHandlerRecorder
}

// Recorder for MockUnaryHandler (not exported)
type _MockUnaryHandlerRecorder struct {
	mock *MockUnaryHandler
}

func NewMockUnaryHandler(ctrl *gomock.Controller) *MockUnaryHandler {
	mock := &MockUnaryHandler{ctrl: ctrl}
	mock.recorder = &_MockUnaryHandlerRecorder{mock}
	return mock
}

func (_m *MockUnaryHandler) EXPECT() *_MockUnaryHandlerRecorder {
	return _m.recorder
}

func (_m *MockUnaryHandler) Handle(_param0 context.Context, _param1 yarpc.ReqMeta, _param2 wire.Value) (Response, error) {
	ret := _m.ctrl.Call(_m, "Handle", _param0, _param1, _param2)
	ret0, _ := ret[0].(Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockUnaryHandlerRecorder) Handle(arg0, arg1, arg2 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Handle", arg0, arg1, arg2)
}
