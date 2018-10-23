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

package yarpcrouter

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpctest"
)

func TestMapRouter(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := NewMapRouter("myservice")

	foo := yarpctest.NewMockUnaryTransportHandler(mockCtrl)
	bar := yarpctest.NewMockUnaryTransportHandler(mockCtrl)
	bazJSON := yarpctest.NewMockUnaryTransportHandler(mockCtrl)
	bazThrift := yarpctest.NewMockUnaryTransportHandler(mockCtrl)
	m.Register([]yarpc.TransportProcedure{
		{
			Name:        "foo",
			HandlerSpec: yarpc.NewUnaryTransportHandlerSpec(foo),
		},
		{
			Name:        "bar",
			Service:     "anotherservice",
			HandlerSpec: yarpc.NewUnaryTransportHandlerSpec(bar),
		},
		{
			Name:        "baz",
			Encoding:    "json",
			HandlerSpec: yarpc.NewUnaryTransportHandlerSpec(bazJSON),
		},
		{
			Name:        "baz",
			Encoding:    "thrift",
			HandlerSpec: yarpc.NewUnaryTransportHandlerSpec(bazThrift),
		},
	})

	tests := []struct {
		service, procedure, encoding string
		want                         yarpc.UnaryTransportHandler
	}{
		{"myservice", "foo", "", foo},
		{"", "foo", "", foo},
		{"anotherservice", "foo", "", nil},
		{"", "bar", "", nil},
		{"myservice", "bar", "", nil},
		{"anotherservice", "bar", "", bar},
		{"myservice", "baz", "thrift", bazThrift},
		{"", "baz", "thrift", bazThrift},
		{"myservice", "baz", "json", bazJSON},
		{"", "baz", "json", bazJSON},
		{"myservice", "baz", "proto", nil},
		{"", "baz", "proto", nil},
		{"unknownservice", "", "", nil},
	}

	for _, tt := range tests {
		got, err := m.Choose(context.Background(), &yarpc.Request{
			Service:   tt.service,
			Procedure: tt.procedure,
			Encoding:  yarpc.Encoding(tt.encoding),
		})
		if tt.want != nil {
			assert.NoError(t, err,
				"Choose(%q, %q, %q) failed", tt.service, tt.procedure, tt.encoding)
			assert.True(t, tt.want == got.Unary(), // want == match, not deep equals
				"Choose(%q, %q, %q) did not match", tt.service, tt.procedure, tt.encoding)
		} else {
			assert.Error(t, err, fmt.Sprintf("Expected error for (%q, %q, %q)", tt.service, tt.procedure, tt.encoding))
		}
	}
}

func TestMapRouter_Procedures(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := NewMapRouter("myservice")

	bar := yarpc.NewUnaryTransportHandlerSpec(yarpctest.NewMockUnaryTransportHandler(mockCtrl))
	foo := yarpc.NewUnaryTransportHandlerSpec(yarpctest.NewMockUnaryTransportHandler(mockCtrl))
	aww := yarpc.NewUnaryTransportHandlerSpec(yarpctest.NewMockUnaryTransportHandler(mockCtrl))
	m.Register([]yarpc.TransportProcedure{
		{
			Name:        "bar",
			Service:     "anotherservice",
			HandlerSpec: bar,
		},
		{
			Name:        "foo",
			Encoding:    "json",
			HandlerSpec: foo,
		},
		{
			Name:        "aww",
			Service:     "anotherservice",
			HandlerSpec: aww,
		},
	})

	expectedOrderedProcedures := []yarpc.TransportProcedure{
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
			Encoding:    "json",
			Service:     "myservice",
			HandlerSpec: foo,
		},
	}

	serviceProcedures := m.Procedures()

	assert.Equal(t, expectedOrderedProcedures, serviceProcedures)
}

func TestEmptyProcedureRegistration(t *testing.T) {
	m := NewMapRouter("test-service-name")

	procedures := []yarpc.TransportProcedure{
		{
			Name:    "",
			Service: "test",
		},
	}

	assert.Panics(t,
		func() { m.Register(procedures) },
		"expected router panic")
}

func IgnoreTestRouterWithMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	req := &yarpc.Request{}
	expectedSpec := yarpc.TransportHandlerSpec{}

	routerMiddleware := yarpctest.NewMockRouterMiddleware(mockCtrl)
	routerMiddleware.EXPECT().Choose(ctx, req, gomock.Any()).Times(1).Return(expectedSpec, nil)

	router := yarpc.ApplyRouteTable(NewMapRouter("service"), routerMiddleware)

	actualSpec, err := router.Choose(ctx, req)

	assert.Equal(t, expectedSpec, actualSpec, "handler spec returned from route table did not match")
	assert.Nil(t, err)
}

func TestProcedureSort(t *testing.T) {
	ps := sortableProcedures{
		{
			Service:  "moe",
			Name:     "echo",
			Encoding: "json",
		},
		{
			Service: "moe",
		},
		{
			Service: "larry",
		},
		{
			Service: "moe",
			Name:    "ping",
		},
		{
			Service:  "moe",
			Name:     "echo",
			Encoding: "raw",
		},
		{
			Service: "moe",
			Name:    "poke",
		},
		{
			Service: "curly",
		},
	}
	sort.Sort(ps)
	assert.Equal(t, sortableProcedures{
		{
			Service: "curly",
		},
		{
			Service: "larry",
		},
		{
			Service: "moe",
		},
		{
			Service:  "moe",
			Name:     "echo",
			Encoding: "json",
		},
		{
			Service:  "moe",
			Name:     "echo",
			Encoding: "raw",
		},
		{
			Service: "moe",
			Name:    "ping",
		},
		{
			Service: "moe",
			Name:    "poke",
		},
	}, ps, "should order procedures lexicographically on (service, procedure, encoding)")
}

func TestUnknownServiceName(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := NewMapRouter("service1")
	foo := yarpctest.NewMockUnaryTransportHandler(mockCtrl)

	m.Register([]yarpc.TransportProcedure{
		{
			Name:        "foo",
			HandlerSpec: yarpc.NewUnaryTransportHandlerSpec(foo),
			Service:     "service2",
			Encoding:    "json",
		},
	})

	_, err := m.Choose(context.Background(), &yarpc.Request{
		Service:   "wrongService",
		Procedure: "foo",
		Encoding:  "json",
	})
	assert.Contains(t, err.Error(), `unrecognized service name "wrongService", available services: "service1", "service2"`)
}
