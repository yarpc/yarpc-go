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

package transport_test

import (
	"testing"

	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMapRegistry(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := transport.NewMapRegistry("myservice")

	foo := transporttest.NewMockHandler(mockCtrl)
	bar := transporttest.NewMockHandler(mockCtrl)
	m.Register([]transport.Registrant{
		{Procedure: "foo", Handler: foo},
		{Service: "anotherservice", Procedure: "bar", Handler: bar},
	})

	tests := []struct {
		service, procedure string
		want               transport.Handler
	}{
		{"myservice", "foo", foo},
		{"", "foo", foo},
		{"anotherservice", "foo", nil},
		{"", "bar", nil},
		{"myservice", "bar", nil},
		{"anotherservice", "bar", bar},
	}

	for _, tt := range tests {
		got, err := m.GetHandler(tt.service, tt.procedure)
		if tt.want != nil {
			assert.NoError(t, err,
				"GetHandler(%q, %q) failed", tt.service, tt.procedure)
			assert.True(t, tt.want == got, // want == match, not deep equals
				"GetHandler(%q, %q) did not match", tt.service, tt.procedure)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestMapRegistry_ServiceProcedures(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	m := transport.NewMapRegistry("myservice")

	bar := transporttest.NewMockHandler(mockCtrl)
	foo := transporttest.NewMockHandler(mockCtrl)
	aww := transporttest.NewMockHandler(mockCtrl)
	m.Register([]transport.Registrant{
		{Service: "anotherservice", Procedure: "bar", Handler: bar},
		{Procedure: "foo", Handler: foo},
		{Service: "anotherservice", Procedure: "aww", Handler: aww},
	})

	expectedOrderedServiceProcedures := []transport.ServiceProcedure{
		{
			Service:   "anotherservice",
			Procedure: "aww",
		},
		{
			Service:   "anotherservice",
			Procedure: "bar",
		},
		{
			Service:   "myservice",
			Procedure: "foo",
		},
	}

	serviceProcedures := m.ServiceProcedures()

	assert.Equal(t, expectedOrderedServiceProcedures, serviceProcedures)
}
