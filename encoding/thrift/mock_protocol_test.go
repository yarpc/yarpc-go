// Automatically generated by MockGen. DO NOT EDIT!
// Source: vendor/go.uber.org/thriftrw/protocol/protocol.go

package thrift

import (
	gomock "github.com/golang/mock/gomock"
	wire "go.uber.org/thriftrw/wire"
	io "io"
)

// Mock of Protocol interface
type MockProtocol struct {
	ctrl     *gomock.Controller
	recorder *_MockProtocolRecorder
}

// Recorder for MockProtocol (not exported)
type _MockProtocolRecorder struct {
	mock *MockProtocol
}

func NewMockProtocol(ctrl *gomock.Controller) *MockProtocol {
	mock := &MockProtocol{ctrl: ctrl}
	mock.recorder = &_MockProtocolRecorder{mock}
	return mock
}

func (_m *MockProtocol) EXPECT() *_MockProtocolRecorder {
	return _m.recorder
}

func (_m *MockProtocol) Encode(v wire.Value, w io.Writer) error {
	ret := _m.ctrl.Call(_m, "Encode", v, w)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockProtocolRecorder) Encode(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Encode", arg0, arg1)
}

func (_m *MockProtocol) EncodeEnveloped(e wire.Envelope, w io.Writer) error {
	ret := _m.ctrl.Call(_m, "EncodeEnveloped", e, w)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockProtocolRecorder) EncodeEnveloped(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "EncodeEnveloped", arg0, arg1)
}

func (_m *MockProtocol) Decode(r io.ReaderAt, t wire.Type) (wire.Value, error) {
	ret := _m.ctrl.Call(_m, "Decode", r, t)
	ret0, _ := ret[0].(wire.Value)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockProtocolRecorder) Decode(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Decode", arg0, arg1)
}

func (_m *MockProtocol) DecodeEnveloped(r io.ReaderAt) (wire.Envelope, error) {
	ret := _m.ctrl.Call(_m, "DecodeEnveloped", r)
	ret0, _ := ret[0].(wire.Envelope)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockProtocolRecorder) DecodeEnveloped(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "DecodeEnveloped", arg0)
}
