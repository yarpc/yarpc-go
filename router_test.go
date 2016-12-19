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

package yarpc_test

import (
	"testing"

	"go.uber.org/yarpc"
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
			ProcedureName: "foo",
			HandlerSpec:   transport.NewUnaryHandlerSpec(foo),
		},
		{
			Service:       "anotherservice",
			ProcedureName: "bar",
			HandlerSpec:   transport.NewUnaryHandlerSpec(bar),
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
		got, err := m.ChooseProcedure(tt.service, tt.procedure)
		if tt.want != nil {
			assert.NoError(t, err,
				"ChooseProcedure(%q, %q) failed", tt.service, tt.procedure)
			assert.True(t, tt.want == got.Unary(), // want == match, not deep equals
				"ChooseProcedure(%q, %q) did not match", tt.service, tt.procedure)
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
			Service:       "anotherservice",
			ProcedureName: "bar",
			HandlerSpec:   bar,
		},
		{
			ProcedureName: "foo",
			HandlerSpec:   foo,
		},
		{
			Service:       "anotherservice",
			ProcedureName: "aww",
			HandlerSpec:   aww,
		},
	})

	expectedOrderedProcedures := []transport.Procedure{
		{
			Service:       "anotherservice",
			ProcedureName: "aww",
			HandlerSpec:   aww,
		},
		{
			Service:       "anotherservice",
			ProcedureName: "bar",
			HandlerSpec:   bar,
		},
		{
			Service:       "myservice",
			ProcedureName: "foo",
			HandlerSpec:   foo,
		},
	}

	serviceProcedures := m.Procedures()

	assert.Equal(t, expectedOrderedProcedures, serviceProcedures)
}

func TestEmptyProcedureRegistration(t *testing.T) {
	m := yarpc.NewMapRouter("test-service-name")

	procedures := []transport.Procedure{
		{
			Service:       "test",
			ProcedureName: "",
		},
	}

	assert.Panics(t,
		func() { m.Register(procedures) },
		"expected router panic")
}
