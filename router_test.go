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

package yarpc_test

import (
	"context"
	"testing"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/middleware/middlewaretest"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMapRouter(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := yarpc.NewMapRouter("myservice")

	foo := transporttest.NewMockUnaryHandler(mockCtrl)
	bar := transporttest.NewMockUnaryHandler(mockCtrl)
	m.Register([]transport.Procedure{
		{
			Name:        "foo",
			HandlerSpec: transport.NewUnaryHandlerSpec(foo),
		},
		{
			Name:        "bar",
			Service:     "anotherservice",
			HandlerSpec: transport.NewUnaryHandlerSpec(bar),
		},
	})

	tests := []struct {
		service, procedure string
		want               transport.UnaryHandler
	}{
		{"myservice", "foo", foo},
		{"", "foo", foo},
		{"anotherservice", "foo", nil},
		{"", "bar", nil},
		{"myservice", "bar", nil},
		{"anotherservice", "bar", bar},
	}

	for _, tt := range tests {
		got, err := m.Choose(context.Background(), &transport.Request{
			Service:   tt.service,
			Procedure: tt.procedure,
		})
		if tt.want != nil {
			assert.NoError(t, err,
				"Choose(%q, %q) failed", tt.service, tt.procedure)
			assert.True(t, tt.want == got.Unary(), // want == match, not deep equals
				"Choose(%q, %q) did not match", tt.service, tt.procedure)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestMapRouter_Procedures(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := yarpc.NewMapRouter("myservice")

	bar := transport.NewUnaryHandlerSpec(transporttest.NewMockUnaryHandler(mockCtrl))
	foo := transport.NewUnaryHandlerSpec(transporttest.NewMockUnaryHandler(mockCtrl))
	aww := transport.NewUnaryHandlerSpec(transporttest.NewMockUnaryHandler(mockCtrl))
	m.Register([]transport.Procedure{
		{
			Name:        "bar",
			Service:     "anotherservice",
			HandlerSpec: bar,
		},
		{
			Name:        "foo",
			HandlerSpec: foo,
		},
		{
			Name:        "aww",
			Service:     "anotherservice",
			HandlerSpec: aww,
		},
	})

	expectedOrderedProcedures := []transport.Procedure{
		{
			Name:        "aww",
			Service:     "anotherservice",
			HandlerSpec: aww,
		},
		{
			Name:        "bar",
			Service:     "anotherservice",
			HandlerSpec: bar,
		},
		{
			Name:        "foo",
			Service:     "myservice",
			HandlerSpec: foo,
		},
	}

	serviceProcedures := m.Procedures()

	assert.Equal(t, expectedOrderedProcedures, serviceProcedures)
}

func TestEmptyProcedureRegistration(t *testing.T) {
	m := yarpc.NewMapRouter("test-service-name")

	procedures := []transport.Procedure{
		{
			Name:    "",
			Service: "test",
		},
	}

	assert.Panics(t,
		func() { m.Register(procedures) },
		"expected router panic")
}

func TestRouterWithMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	req := &transport.Request{}
	expectedSpec := transport.HandlerSpec{}

	routerMiddleware := middlewaretest.NewMockRouter(mockCtrl)
	routerMiddleware.EXPECT().Choose(ctx, req, gomock.Any()).Times(1).Return(expectedSpec, nil)

	router := middleware.ApplyRouteTable(yarpc.NewMapRouter("service"), routerMiddleware)

	actualSpec, err := router.Choose(ctx, req)

	assert.Equal(t, expectedSpec, actualSpec, "handler spec returned from route table did not match")
	assert.Nil(t, err)
}
