// Code generated by thriftrw-plugin-yarpc
// @generated

package bartest

import (
	"context"
	"github.com/golang/mock/gomock"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/extends/barclient"
)

// MockClient implements a gomock-compatible mock client for service
// Bar.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *_MockClientRecorder
}

var _ barclient.Interface = (*MockClient)(nil)

type _MockClientRecorder struct {
	mock *MockClient
}

// Build a new mock client for service Bar.
//
// 	mockCtrl := gomock.NewController(t)
// 	client := bartest.NewMockClient(mockCtrl)
//
// Use EXPECT() to set expectations on the mock.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &_MockClientRecorder{mock}
	return mock
}

// EXPECT returns an object that allows you to define an expectation on the
// Bar mock client.
func (m *MockClient) EXPECT() *_MockClientRecorder {
	return m.recorder
}

// Name responds to a Name call based on the mock expectations. This
// call will fail if the mock does not expect this call. Use EXPECT to expect
// a call to this function.
//
// 	client.EXPECT().Name(gomock.Any(), ...).Return(...)
// 	... := client.Name(...)
func (m *MockClient) Name(
	ctx context.Context,
	opts ...yarpc.CallOption,
) (success string, err error) {

	args := []interface{}{ctx}
	for _, o := range opts {
		args = append(args, o)
	}
	i := 0
	ret := m.ctrl.Call(m, "Name", args...)
	success, _ = ret[i].(string)
	i++
	err, _ = ret[i].(error)
	return
}

func (mr *_MockClientRecorder) Name(
	ctx interface{},
	opts ...interface{},
) *gomock.Call {
	args := append([]interface{}{ctx}, opts...)
	return mr.mock.ctrl.RecordCall(mr.mock, "Name", args...)
}
