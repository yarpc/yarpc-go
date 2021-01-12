// Code generated by thriftrw-plugin-yarpc
// @generated

// Copyright (c) 2021 Uber Technologies, Inc.
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

package testservicetest

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	yarpc "go.uber.org/yarpc"
	testserviceclient "go.uber.org/yarpc/encoding/thrift/internal/observabilitytest/test/testserviceclient"
)

// MockClient implements a gomock-compatible mock client for service
// TestService.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *_MockClientRecorder
}

var _ testserviceclient.Interface = (*MockClient)(nil)

type _MockClientRecorder struct {
	mock *MockClient
}

// Build a new mock client for service TestService.
//
// 	mockCtrl := gomock.NewController(t)
// 	client := testservicetest.NewMockClient(mockCtrl)
//
// Use EXPECT() to set expectations on the mock.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &_MockClientRecorder{mock}
	return mock
}

// EXPECT returns an object that allows you to define an expectation on the
// TestService mock client.
func (m *MockClient) EXPECT() *_MockClientRecorder {
	return m.recorder
}

// Call responds to a Call call based on the mock expectations. This
// call will fail if the mock does not expect this call. Use EXPECT to expect
// a call to this function.
//
// 	client.EXPECT().Call(gomock.Any(), ...).Return(...)
// 	... := client.Call(...)
func (m *MockClient) Call(
	ctx context.Context,
	_Key string,
	opts ...yarpc.CallOption,
) (success string, err error) {

	args := []interface{}{ctx, _Key}
	for _, o := range opts {
		args = append(args, o)
	}
	i := 0
	ret := m.ctrl.Call(m, "Call", args...)
	success, _ = ret[i].(string)
	i++
	err, _ = ret[i].(error)
	return
}

func (mr *_MockClientRecorder) Call(
	ctx interface{},
	_Key interface{},
	opts ...interface{},
) *gomock.Call {
	args := append([]interface{}{ctx, _Key}, opts...)
	return mr.mock.ctrl.RecordCall(mr.mock, "Call", args...)
}
