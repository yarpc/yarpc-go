// Copyright (c) 2026 Uber Technologies, Inc.
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

package thrift

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/procedure"
)

func TestBuildProcedures_propagatesExceptions(t *testing.T) {
	t.Parallel()

	exceptions := map[string]string{
		"KeyDoesNotExist": "INVALID_ARGUMENT",
		"Other":           "__not_set__",
	}

	svc := Service{
		Name: "MyService",
		Methods: []Method{
			{
				Name: "myMethod",
				HandlerSpec: HandlerSpec{
					Type: transport.Unary,
					Unary: func(context.Context, wire.Value) (Response, error) {
						return Response{}, nil
					},
				},
				Signature:  "MyMethod()",
				Exceptions: exceptions,
			},
		},
	}

	procs := BuildProcedures(svc, NoWire(false))
	require.Len(t, procs, 1)

	assert.Equal(t, procedure.ToName("MyService", "myMethod"), procs[0].Name)
	assert.Equal(t, exceptions, procs[0].Exceptions)
}

func TestBuildProcedures_nilExceptions(t *testing.T) {
	t.Parallel()

	svc := Service{
		Name: "MyService",
		Methods: []Method{
			{
				Name: "myMethod",
				HandlerSpec: HandlerSpec{
					Type: transport.Unary,
					Unary: func(context.Context, wire.Value) (Response, error) {
						return Response{}, nil
					},
				},
				Signature: "MyMethod()",
			},
		},
	}

	procs := BuildProcedures(svc, NoWire(false))
	require.Len(t, procs, 1)
	assert.Nil(t, procs[0].Exceptions)
}
