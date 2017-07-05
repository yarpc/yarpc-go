// Code generated by thriftrw-plugin-yarpc
// @generated

// Copyright (c) 2017 Uber Technologies, Inc.
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

package hellotest

import (
	"context"
	"github.com/golang/mock/gomock"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/examples/thrift-oneway/sink"
	"go.uber.org/yarpc/internal/examples/thrift-oneway/sink/helloclient"
)

// MockClient implements a gomock-compatible mock client for service
// Hello.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *_MockClientRecorder
}

var _ helloclient.Interface = (*MockClient)(nil)

type _MockClientRecorder struct {
	mock *MockClient
}

// Build a new mock client for service Hello.
//
// 	mockCtrl := gomock.NewController(t)
// 	client := hellotest.NewMockClient(mockCtrl)
//
// Use EXPECT() to set expectations on the mock.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &_MockClientRecorder{mock}
	return mock
}

// EXPECT returns an object that allows you to define an expectation on the
// Hello mock client.
func (m *MockClient) EXPECT() *_MockClientRecorder {
	return m.recorder
}

// Sink responds to a Sink call based on the mock expectations. This
// call will fail if the mock does not expect this call. Use EXPECT to expect
// a call to this function.
//
// 	client.EXPECT().Sink(gomock.Any(), ...).Return(...)
// 	... := client.Sink(...)
func (m *MockClient) Sink(
	ctx context.Context,
	_Snk *sink.SinkRequest,
	opts ...yarpc.CallOption,
) (ack yarpc.Ack, err error) {

	args := []interface{}{ctx, _Snk}
	for _, o := range opts {
		args = append(args, o)
	}
	i := 0
	ret := m.ctrl.Call(m, "Sink", args...)
	ack, _ = ret[i].(yarpc.Ack)
	i++
	err, _ = ret[i].(error)
	return
}

func (mr *_MockClientRecorder) Sink(
	ctx interface{},
	_Snk interface{},
	opts ...interface{},
) *gomock.Call {
	args := append([]interface{}{ctx, _Snk}, opts...)
	return mr.mock.ctrl.RecordCall(mr.mock, "Sink", args...)
}
