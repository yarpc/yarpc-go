// Code generated by MockGen. DO NOT EDIT.
// Source: internal/interceptor/outbound.go

// Package interceptortest is a generated GoMock package.
package interceptortest

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	transport "go.uber.org/yarpc/api/transport"
	interceptor "go.uber.org/yarpc/internal/interceptor"
)

// MockUnaryOutbound is a mock of UnaryOutbound interface.
type MockUnaryOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockUnaryOutboundMockRecorder
}

// MockUnaryOutboundMockRecorder is the mock recorder for MockUnaryOutbound.
type MockUnaryOutboundMockRecorder struct {
	mock *MockUnaryOutbound
}

// NewMockUnaryOutbound creates a new mock instance.
func NewMockUnaryOutbound(ctrl *gomock.Controller) *MockUnaryOutbound {
	mock := &MockUnaryOutbound{ctrl: ctrl}
	mock.recorder = &MockUnaryOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUnaryOutbound) EXPECT() *MockUnaryOutboundMockRecorder {
	return m.recorder
}

// Call mocks base method.
func (m *MockUnaryOutbound) Call(ctx context.Context, request *transport.Request, out interceptor.UnchainedUnaryOutbound) (*transport.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Call", ctx, request, out)
	ret0, _ := ret[0].(*transport.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Call indicates an expected call of Call.
func (mr *MockUnaryOutboundMockRecorder) Call(ctx, request, out interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Call", reflect.TypeOf((*MockUnaryOutbound)(nil).Call), ctx, request, out)
}

// MockOnewayOutbound is a mock of OnewayOutbound interface.
type MockOnewayOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockOnewayOutboundMockRecorder
}

// MockOnewayOutboundMockRecorder is the mock recorder for MockOnewayOutbound.
type MockOnewayOutboundMockRecorder struct {
	mock *MockOnewayOutbound
}

// NewMockOnewayOutbound creates a new mock instance.
func NewMockOnewayOutbound(ctrl *gomock.Controller) *MockOnewayOutbound {
	mock := &MockOnewayOutbound{ctrl: ctrl}
	mock.recorder = &MockOnewayOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOnewayOutbound) EXPECT() *MockOnewayOutboundMockRecorder {
	return m.recorder
}

// CallOneway mocks base method.
func (m *MockOnewayOutbound) CallOneway(ctx context.Context, request *transport.Request, out interceptor.UnchainedOnewayOutbound) (transport.Ack, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CallOneway", ctx, request, out)
	ret0, _ := ret[0].(transport.Ack)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CallOneway indicates an expected call of CallOneway.
func (mr *MockOnewayOutboundMockRecorder) CallOneway(ctx, request, out interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CallOneway", reflect.TypeOf((*MockOnewayOutbound)(nil).CallOneway), ctx, request, out)
}

// MockStreamOutbound is a mock of StreamOutbound interface.
type MockStreamOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockStreamOutboundMockRecorder
}

// MockStreamOutboundMockRecorder is the mock recorder for MockStreamOutbound.
type MockStreamOutboundMockRecorder struct {
	mock *MockStreamOutbound
}

// NewMockStreamOutbound creates a new mock instance.
func NewMockStreamOutbound(ctrl *gomock.Controller) *MockStreamOutbound {
	mock := &MockStreamOutbound{ctrl: ctrl}
	mock.recorder = &MockStreamOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamOutbound) EXPECT() *MockStreamOutboundMockRecorder {
	return m.recorder
}

// CallStream mocks base method.
func (m *MockStreamOutbound) CallStream(ctx context.Context, req *transport.StreamRequest, out interceptor.UnchainedStreamOutbound) (*transport.ClientStream, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CallStream", ctx, req, out)
	ret0, _ := ret[0].(*transport.ClientStream)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CallStream indicates an expected call of CallStream.
func (mr *MockStreamOutboundMockRecorder) CallStream(ctx, req, out interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CallStream", reflect.TypeOf((*MockStreamOutbound)(nil).CallStream), ctx, req, out)
}

// MockOutbound is a mock of Outbound interface.
type MockOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockOutboundMockRecorder
}

// MockOutboundMockRecorder is the mock recorder for MockOutbound.
type MockOutboundMockRecorder struct {
	mock *MockOutbound
}

// NewMockOutbound creates a new mock instance.
func NewMockOutbound(ctrl *gomock.Controller) *MockOutbound {
	mock := &MockOutbound{ctrl: ctrl}
	mock.recorder = &MockOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOutbound) EXPECT() *MockOutboundMockRecorder {
	return m.recorder
}

// IsRunning mocks base method.
func (m *MockOutbound) IsRunning() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsRunning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsRunning indicates an expected call of IsRunning.
func (mr *MockOutboundMockRecorder) IsRunning() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsRunning", reflect.TypeOf((*MockOutbound)(nil).IsRunning))
}

// Start mocks base method.
func (m *MockOutbound) Start() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockOutboundMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockOutbound)(nil).Start))
}

// Stop mocks base method.
func (m *MockOutbound) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockOutboundMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockOutbound)(nil).Stop))
}

// Transports mocks base method.
func (m *MockOutbound) Transports() []transport.Transport {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Transports")
	ret0, _ := ret[0].([]transport.Transport)
	return ret0
}

// Transports indicates an expected call of Transports.
func (mr *MockOutboundMockRecorder) Transports() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Transports", reflect.TypeOf((*MockOutbound)(nil).Transports))
}

// MockUnchainedUnaryOutbound is a mock of UnchainedUnaryOutbound interface.
type MockUnchainedUnaryOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockUnchainedUnaryOutboundMockRecorder
}

// MockUnchainedUnaryOutboundMockRecorder is the mock recorder for MockUnchainedUnaryOutbound.
type MockUnchainedUnaryOutboundMockRecorder struct {
	mock *MockUnchainedUnaryOutbound
}

// NewMockUnchainedUnaryOutbound creates a new mock instance.
func NewMockUnchainedUnaryOutbound(ctrl *gomock.Controller) *MockUnchainedUnaryOutbound {
	mock := &MockUnchainedUnaryOutbound{ctrl: ctrl}
	mock.recorder = &MockUnchainedUnaryOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUnchainedUnaryOutbound) EXPECT() *MockUnchainedUnaryOutboundMockRecorder {
	return m.recorder
}

// IsRunning mocks base method.
func (m *MockUnchainedUnaryOutbound) IsRunning() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsRunning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsRunning indicates an expected call of IsRunning.
func (mr *MockUnchainedUnaryOutboundMockRecorder) IsRunning() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsRunning", reflect.TypeOf((*MockUnchainedUnaryOutbound)(nil).IsRunning))
}

// Start mocks base method.
func (m *MockUnchainedUnaryOutbound) Start() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockUnchainedUnaryOutboundMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockUnchainedUnaryOutbound)(nil).Start))
}

// Stop mocks base method.
func (m *MockUnchainedUnaryOutbound) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockUnchainedUnaryOutboundMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockUnchainedUnaryOutbound)(nil).Stop))
}

// Transports mocks base method.
func (m *MockUnchainedUnaryOutbound) Transports() []transport.Transport {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Transports")
	ret0, _ := ret[0].([]transport.Transport)
	return ret0
}

// Transports indicates an expected call of Transports.
func (mr *MockUnchainedUnaryOutboundMockRecorder) Transports() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Transports", reflect.TypeOf((*MockUnchainedUnaryOutbound)(nil).Transports))
}

// UnchainedCall mocks base method.
func (m *MockUnchainedUnaryOutbound) UnchainedCall(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnchainedCall", ctx, request)
	ret0, _ := ret[0].(*transport.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UnchainedCall indicates an expected call of UnchainedCall.
func (mr *MockUnchainedUnaryOutboundMockRecorder) UnchainedCall(ctx, request interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnchainedCall", reflect.TypeOf((*MockUnchainedUnaryOutbound)(nil).UnchainedCall), ctx, request)
}

// MockUnchainedOnewayOutbound is a mock of UnchainedOnewayOutbound interface.
type MockUnchainedOnewayOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockUnchainedOnewayOutboundMockRecorder
}

// MockUnchainedOnewayOutboundMockRecorder is the mock recorder for MockUnchainedOnewayOutbound.
type MockUnchainedOnewayOutboundMockRecorder struct {
	mock *MockUnchainedOnewayOutbound
}

// NewMockUnchainedOnewayOutbound creates a new mock instance.
func NewMockUnchainedOnewayOutbound(ctrl *gomock.Controller) *MockUnchainedOnewayOutbound {
	mock := &MockUnchainedOnewayOutbound{ctrl: ctrl}
	mock.recorder = &MockUnchainedOnewayOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUnchainedOnewayOutbound) EXPECT() *MockUnchainedOnewayOutboundMockRecorder {
	return m.recorder
}

// IsRunning mocks base method.
func (m *MockUnchainedOnewayOutbound) IsRunning() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsRunning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsRunning indicates an expected call of IsRunning.
func (mr *MockUnchainedOnewayOutboundMockRecorder) IsRunning() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsRunning", reflect.TypeOf((*MockUnchainedOnewayOutbound)(nil).IsRunning))
}

// Start mocks base method.
func (m *MockUnchainedOnewayOutbound) Start() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockUnchainedOnewayOutboundMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockUnchainedOnewayOutbound)(nil).Start))
}

// Stop mocks base method.
func (m *MockUnchainedOnewayOutbound) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockUnchainedOnewayOutboundMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockUnchainedOnewayOutbound)(nil).Stop))
}

// Transports mocks base method.
func (m *MockUnchainedOnewayOutbound) Transports() []transport.Transport {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Transports")
	ret0, _ := ret[0].([]transport.Transport)
	return ret0
}

// Transports indicates an expected call of Transports.
func (mr *MockUnchainedOnewayOutboundMockRecorder) Transports() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Transports", reflect.TypeOf((*MockUnchainedOnewayOutbound)(nil).Transports))
}

// UnchainedOnewayCall mocks base method.
func (m *MockUnchainedOnewayOutbound) UnchainedOnewayCall(ctx context.Context, request *transport.Request) (transport.Ack, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnchainedOnewayCall", ctx, request)
	ret0, _ := ret[0].(transport.Ack)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UnchainedOnewayCall indicates an expected call of UnchainedOnewayCall.
func (mr *MockUnchainedOnewayOutboundMockRecorder) UnchainedOnewayCall(ctx, request interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnchainedOnewayCall", reflect.TypeOf((*MockUnchainedOnewayOutbound)(nil).UnchainedOnewayCall), ctx, request)
}

// MockUnchainedStreamOutbound is a mock of UnchainedStreamOutbound interface.
type MockUnchainedStreamOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockUnchainedStreamOutboundMockRecorder
}

// MockUnchainedStreamOutboundMockRecorder is the mock recorder for MockUnchainedStreamOutbound.
type MockUnchainedStreamOutboundMockRecorder struct {
	mock *MockUnchainedStreamOutbound
}

// NewMockUnchainedStreamOutbound creates a new mock instance.
func NewMockUnchainedStreamOutbound(ctrl *gomock.Controller) *MockUnchainedStreamOutbound {
	mock := &MockUnchainedStreamOutbound{ctrl: ctrl}
	mock.recorder = &MockUnchainedStreamOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUnchainedStreamOutbound) EXPECT() *MockUnchainedStreamOutboundMockRecorder {
	return m.recorder
}

// IsRunning mocks base method.
func (m *MockUnchainedStreamOutbound) IsRunning() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsRunning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsRunning indicates an expected call of IsRunning.
func (mr *MockUnchainedStreamOutboundMockRecorder) IsRunning() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsRunning", reflect.TypeOf((*MockUnchainedStreamOutbound)(nil).IsRunning))
}

// Start mocks base method.
func (m *MockUnchainedStreamOutbound) Start() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockUnchainedStreamOutboundMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockUnchainedStreamOutbound)(nil).Start))
}

// Stop mocks base method.
func (m *MockUnchainedStreamOutbound) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockUnchainedStreamOutboundMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockUnchainedStreamOutbound)(nil).Stop))
}

// Transports mocks base method.
func (m *MockUnchainedStreamOutbound) Transports() []transport.Transport {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Transports")
	ret0, _ := ret[0].([]transport.Transport)
	return ret0
}

// Transports indicates an expected call of Transports.
func (mr *MockUnchainedStreamOutboundMockRecorder) Transports() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Transports", reflect.TypeOf((*MockUnchainedStreamOutbound)(nil).Transports))
}

// UnchainedStreamCall mocks base method.
func (m *MockUnchainedStreamOutbound) UnchainedStreamCall(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnchainedStreamCall", ctx, req)
	ret0, _ := ret[0].(*transport.ClientStream)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UnchainedStreamCall indicates an expected call of UnchainedStreamCall.
func (mr *MockUnchainedStreamOutboundMockRecorder) UnchainedStreamCall(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnchainedStreamCall", reflect.TypeOf((*MockUnchainedStreamOutbound)(nil).UnchainedStreamCall), ctx, req)
}
