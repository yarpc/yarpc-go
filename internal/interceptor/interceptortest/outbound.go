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

// MockUnaryOutboundChain is a mock of UnaryOutboundChain interface.
type MockUnaryOutboundChain struct {
	ctrl     *gomock.Controller
	recorder *MockUnaryOutboundChainMockRecorder
}

// MockUnaryOutboundChainMockRecorder is the mock recorder for MockUnaryOutboundChain.
type MockUnaryOutboundChainMockRecorder struct {
	mock *MockUnaryOutboundChain
}

// NewMockUnaryOutboundChain creates a new mock instance.
func NewMockUnaryOutboundChain(ctrl *gomock.Controller) *MockUnaryOutboundChain {
	mock := &MockUnaryOutboundChain{ctrl: ctrl}
	mock.recorder = &MockUnaryOutboundChainMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUnaryOutboundChain) EXPECT() *MockUnaryOutboundChainMockRecorder {
	return m.recorder
}

// Next mocks base method.
func (m *MockUnaryOutboundChain) Next(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Next", ctx, request)
	ret0, _ := ret[0].(*transport.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Next indicates an expected call of Next.
func (mr *MockUnaryOutboundChainMockRecorder) Next(ctx, request interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Next", reflect.TypeOf((*MockUnaryOutboundChain)(nil).Next), ctx, request)
}

// Outbound mocks base method.
func (m *MockUnaryOutboundChain) Outbound() interceptor.Outbound {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Outbound")
	ret0, _ := ret[0].(interceptor.Outbound)
	return ret0
}

// Outbound indicates an expected call of Outbound.
func (mr *MockUnaryOutboundChainMockRecorder) Outbound() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Outbound", reflect.TypeOf((*MockUnaryOutboundChain)(nil).Outbound))
}

// MockOnewayOutboundChain is a mock of OnewayOutboundChain interface.
type MockOnewayOutboundChain struct {
	ctrl     *gomock.Controller
	recorder *MockOnewayOutboundChainMockRecorder
}

// MockOnewayOutboundChainMockRecorder is the mock recorder for MockOnewayOutboundChain.
type MockOnewayOutboundChainMockRecorder struct {
	mock *MockOnewayOutboundChain
}

// NewMockOnewayOutboundChain creates a new mock instance.
func NewMockOnewayOutboundChain(ctrl *gomock.Controller) *MockOnewayOutboundChain {
	mock := &MockOnewayOutboundChain{ctrl: ctrl}
	mock.recorder = &MockOnewayOutboundChainMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOnewayOutboundChain) EXPECT() *MockOnewayOutboundChainMockRecorder {
	return m.recorder
}

// Next mocks base method.
func (m *MockOnewayOutboundChain) Next(ctx context.Context, request *transport.Request) (transport.Ack, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Next", ctx, request)
	ret0, _ := ret[0].(transport.Ack)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Next indicates an expected call of Next.
func (mr *MockOnewayOutboundChainMockRecorder) Next(ctx, request interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Next", reflect.TypeOf((*MockOnewayOutboundChain)(nil).Next), ctx, request)
}

// Outbound mocks base method.
func (m *MockOnewayOutboundChain) Outbound() interceptor.Outbound {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Outbound")
	ret0, _ := ret[0].(interceptor.Outbound)
	return ret0
}

// Outbound indicates an expected call of Outbound.
func (mr *MockOnewayOutboundChainMockRecorder) Outbound() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Outbound", reflect.TypeOf((*MockOnewayOutboundChain)(nil).Outbound))
}

// MockStreamOutboundChain is a mock of StreamOutboundChain interface.
type MockStreamOutboundChain struct {
	ctrl     *gomock.Controller
	recorder *MockStreamOutboundChainMockRecorder
}

// MockStreamOutboundChainMockRecorder is the mock recorder for MockStreamOutboundChain.
type MockStreamOutboundChainMockRecorder struct {
	mock *MockStreamOutboundChain
}

// NewMockStreamOutboundChain creates a new mock instance.
func NewMockStreamOutboundChain(ctrl *gomock.Controller) *MockStreamOutboundChain {
	mock := &MockStreamOutboundChain{ctrl: ctrl}
	mock.recorder = &MockStreamOutboundChainMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamOutboundChain) EXPECT() *MockStreamOutboundChainMockRecorder {
	return m.recorder
}

// Next mocks base method.
func (m *MockStreamOutboundChain) Next(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Next", ctx, request)
	ret0, _ := ret[0].(*transport.ClientStream)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Next indicates an expected call of Next.
func (mr *MockStreamOutboundChainMockRecorder) Next(ctx, request interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Next", reflect.TypeOf((*MockStreamOutboundChain)(nil).Next), ctx, request)
}

// Outbound mocks base method.
func (m *MockStreamOutboundChain) Outbound() interceptor.Outbound {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Outbound")
	ret0, _ := ret[0].(interceptor.Outbound)
	return ret0
}

// Outbound indicates an expected call of Outbound.
func (mr *MockStreamOutboundChainMockRecorder) Outbound() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Outbound", reflect.TypeOf((*MockStreamOutboundChain)(nil).Outbound))
}

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
func (m *MockUnaryOutbound) Call(ctx context.Context, request *transport.Request, out interceptor.UnaryOutboundChain) (*transport.Response, error) {
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
func (m *MockOnewayOutbound) CallOneway(ctx context.Context, request *transport.Request, out interceptor.OnewayOutboundChain) (transport.Ack, error) {
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
func (m *MockStreamOutbound) CallStream(ctx context.Context, req *transport.StreamRequest, out interceptor.StreamOutboundChain) (*transport.ClientStream, error) {
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

// MockDirectUnaryOutbound is a mock of DirectUnaryOutbound interface.
type MockDirectUnaryOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockDirectUnaryOutboundMockRecorder
}

// MockDirectUnaryOutboundMockRecorder is the mock recorder for MockDirectUnaryOutbound.
type MockDirectUnaryOutboundMockRecorder struct {
	mock *MockDirectUnaryOutbound
}

// NewMockDirectUnaryOutbound creates a new mock instance.
func NewMockDirectUnaryOutbound(ctrl *gomock.Controller) *MockDirectUnaryOutbound {
	mock := &MockDirectUnaryOutbound{ctrl: ctrl}
	mock.recorder = &MockDirectUnaryOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDirectUnaryOutbound) EXPECT() *MockDirectUnaryOutboundMockRecorder {
	return m.recorder
}

// DirectCall mocks base method.
func (m *MockDirectUnaryOutbound) DirectCall(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DirectCall", ctx, request)
	ret0, _ := ret[0].(*transport.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DirectCall indicates an expected call of DirectCall.
func (mr *MockDirectUnaryOutboundMockRecorder) DirectCall(ctx, request interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DirectCall", reflect.TypeOf((*MockDirectUnaryOutbound)(nil).DirectCall), ctx, request)
}

// IsRunning mocks base method.
func (m *MockDirectUnaryOutbound) IsRunning() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsRunning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsRunning indicates an expected call of IsRunning.
func (mr *MockDirectUnaryOutboundMockRecorder) IsRunning() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsRunning", reflect.TypeOf((*MockDirectUnaryOutbound)(nil).IsRunning))
}

// Start mocks base method.
func (m *MockDirectUnaryOutbound) Start() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockDirectUnaryOutboundMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockDirectUnaryOutbound)(nil).Start))
}

// Stop mocks base method.
func (m *MockDirectUnaryOutbound) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockDirectUnaryOutboundMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockDirectUnaryOutbound)(nil).Stop))
}

// Transports mocks base method.
func (m *MockDirectUnaryOutbound) Transports() []transport.Transport {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Transports")
	ret0, _ := ret[0].([]transport.Transport)
	return ret0
}

// Transports indicates an expected call of Transports.
func (mr *MockDirectUnaryOutboundMockRecorder) Transports() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Transports", reflect.TypeOf((*MockDirectUnaryOutbound)(nil).Transports))
}

// MockDirectOnewayOutbound is a mock of DirectOnewayOutbound interface.
type MockDirectOnewayOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockDirectOnewayOutboundMockRecorder
}

// MockDirectOnewayOutboundMockRecorder is the mock recorder for MockDirectOnewayOutbound.
type MockDirectOnewayOutboundMockRecorder struct {
	mock *MockDirectOnewayOutbound
}

// NewMockDirectOnewayOutbound creates a new mock instance.
func NewMockDirectOnewayOutbound(ctrl *gomock.Controller) *MockDirectOnewayOutbound {
	mock := &MockDirectOnewayOutbound{ctrl: ctrl}
	mock.recorder = &MockDirectOnewayOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDirectOnewayOutbound) EXPECT() *MockDirectOnewayOutboundMockRecorder {
	return m.recorder
}

// DirectCallOneway mocks base method.
func (m *MockDirectOnewayOutbound) DirectCallOneway(ctx context.Context, request *transport.Request) (transport.Ack, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DirectCallOneway", ctx, request)
	ret0, _ := ret[0].(transport.Ack)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DirectCallOneway indicates an expected call of DirectCallOneway.
func (mr *MockDirectOnewayOutboundMockRecorder) DirectCallOneway(ctx, request interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DirectCallOneway", reflect.TypeOf((*MockDirectOnewayOutbound)(nil).DirectCallOneway), ctx, request)
}

// IsRunning mocks base method.
func (m *MockDirectOnewayOutbound) IsRunning() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsRunning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsRunning indicates an expected call of IsRunning.
func (mr *MockDirectOnewayOutboundMockRecorder) IsRunning() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsRunning", reflect.TypeOf((*MockDirectOnewayOutbound)(nil).IsRunning))
}

// Start mocks base method.
func (m *MockDirectOnewayOutbound) Start() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockDirectOnewayOutboundMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockDirectOnewayOutbound)(nil).Start))
}

// Stop mocks base method.
func (m *MockDirectOnewayOutbound) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockDirectOnewayOutboundMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockDirectOnewayOutbound)(nil).Stop))
}

// Transports mocks base method.
func (m *MockDirectOnewayOutbound) Transports() []transport.Transport {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Transports")
	ret0, _ := ret[0].([]transport.Transport)
	return ret0
}

// Transports indicates an expected call of Transports.
func (mr *MockDirectOnewayOutboundMockRecorder) Transports() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Transports", reflect.TypeOf((*MockDirectOnewayOutbound)(nil).Transports))
}

// MockDirectStreamOutbound is a mock of DirectStreamOutbound interface.
type MockDirectStreamOutbound struct {
	ctrl     *gomock.Controller
	recorder *MockDirectStreamOutboundMockRecorder
}

// MockDirectStreamOutboundMockRecorder is the mock recorder for MockDirectStreamOutbound.
type MockDirectStreamOutboundMockRecorder struct {
	mock *MockDirectStreamOutbound
}

// NewMockDirectStreamOutbound creates a new mock instance.
func NewMockDirectStreamOutbound(ctrl *gomock.Controller) *MockDirectStreamOutbound {
	mock := &MockDirectStreamOutbound{ctrl: ctrl}
	mock.recorder = &MockDirectStreamOutboundMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDirectStreamOutbound) EXPECT() *MockDirectStreamOutboundMockRecorder {
	return m.recorder
}

// DirectCallStream mocks base method.
func (m *MockDirectStreamOutbound) DirectCallStream(ctx context.Context, req *transport.StreamRequest) (*transport.ClientStream, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DirectCallStream", ctx, req)
	ret0, _ := ret[0].(*transport.ClientStream)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DirectCallStream indicates an expected call of DirectCallStream.
func (mr *MockDirectStreamOutboundMockRecorder) DirectCallStream(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DirectCallStream", reflect.TypeOf((*MockDirectStreamOutbound)(nil).DirectCallStream), ctx, req)
}

// IsRunning mocks base method.
func (m *MockDirectStreamOutbound) IsRunning() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsRunning")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsRunning indicates an expected call of IsRunning.
func (mr *MockDirectStreamOutboundMockRecorder) IsRunning() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsRunning", reflect.TypeOf((*MockDirectStreamOutbound)(nil).IsRunning))
}

// Start mocks base method.
func (m *MockDirectStreamOutbound) Start() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockDirectStreamOutboundMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockDirectStreamOutbound)(nil).Start))
}

// Stop mocks base method.
func (m *MockDirectStreamOutbound) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockDirectStreamOutboundMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockDirectStreamOutbound)(nil).Stop))
}

// Transports mocks base method.
func (m *MockDirectStreamOutbound) Transports() []transport.Transport {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Transports")
	ret0, _ := ret[0].([]transport.Transport)
	return ret0
}

// Transports indicates an expected call of Transports.
func (mr *MockDirectStreamOutboundMockRecorder) Transports() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Transports", reflect.TypeOf((*MockDirectStreamOutbound)(nil).Transports))
}