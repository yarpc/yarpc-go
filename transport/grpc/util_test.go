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

package grpc

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcedureNameFunctionsBidirectional(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		ProcedureName string
		ServiceName   string
		MethodName    string
		FullMethod    string
	}{
		{
			ProcedureName: "KeyValue::GetValue",
			ServiceName:   "KeyValue",
			MethodName:    "GetValue",
			FullMethod:    "/KeyValue/GetValue",
		},
		{
			ProcedureName: "foo",
			ServiceName:   "__default__",
			MethodName:    "foo",
			FullMethod:    "/__default__/foo",
		},
		{
			ProcedureName: "foo/bar",
			ServiceName:   "__default__",
			MethodName:    url.QueryEscape("foo/bar"),
			FullMethod:    "/__default__/" + url.QueryEscape("foo/bar"),
		},
	} {
		t.Run(tt.ProcedureName, func(t *testing.T) {
			serviceName, methodName, err := procedureNameToServiceNameMethodName(tt.ProcedureName)
			assert.NoError(t, err)
			assert.Equal(t, tt.ServiceName, serviceName)
			assert.Equal(t, tt.MethodName, methodName)
			procedureName, err := procedureToName(serviceName, methodName)
			assert.NoError(t, err)
			assert.Equal(t, tt.ProcedureName, procedureName)
			assert.Equal(t, tt.FullMethod, toFullMethod(serviceName, methodName))
		})
	}
}

func TestProcedureNameEmpty(t *testing.T) {
	_, _, err := procedureNameToServiceNameMethodName("")
	require.Error(t, err)
	_, err = procedureNameToFullMethod("")
	require.Error(t, err)
}

func TestToFullMethodEmptyServiceName(t *testing.T) {
	require.Equal(t, "/__default__/foo", toFullMethod("", "foo"))
}

func TestProcedureToNameInvalidNames(t *testing.T) {
	_, err := procedureToName("%%A", "foo")
	require.Error(t, err)
	_, err = procedureToName("foo", "%%A")
	require.Error(t, err)
}
