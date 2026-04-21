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

package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/plugin/api"
	"go.uber.org/yarpc/api/transport"
)

// expectedExceptionsMap builds the same shape as methodExceptionsMapLiteral for assertions.
func expectedExceptionsMap(entries ...string) string {
	if len(entries) == 0 {
		return "nil"
	}
	return "map[string]string{" + strings.Join(entries, ", ") + "}"
}

func TestExceptionTypeReference(t *testing.T) {
	t.Parallel()
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, exceptionTypeReference(nil))
	})
	t.Run("empty type", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, exceptionTypeReference(&api.Type{}))
	})
	t.Run("pointer to empty type", func(t *testing.T) {
		t.Parallel()
		// Unwraps one pointer level, then hits break (no ReferenceType, no further PointerType).
		assert.Nil(t, exceptionTypeReference(&api.Type{
			PointerType: &api.Type{},
		}))
	})
	t.Run("direct reference", func(t *testing.T) {
		t.Parallel()
		ref := &api.TypeReference{Name: "E"}
		assert.Same(t, ref, exceptionTypeReference(&api.Type{ReferenceType: ref}))
	})
	t.Run("pointer to reference", func(t *testing.T) {
		t.Parallel()
		ref := &api.TypeReference{Name: "E"}
		assert.Same(t, ref, exceptionTypeReference(&api.Type{
			PointerType: &api.Type{ReferenceType: ref},
		}))
	})
}

// exceptionArg is a throws-list entry: ThriftRW models exceptions as a pointer to a named type.
func exceptionArg(typeName string, annotations map[string]string) *api.Argument {
	return exceptionArgWithImport("", typeName, annotations)
}

func exceptionArgWithImport(exceptionImportPath, typeName string, annotations map[string]string) *api.Argument {
	return &api.Argument{
		Type: &api.Type{
			PointerType: &api.Type{
				ReferenceType: &api.TypeReference{
					Name:        typeName,
					ImportPath:  exceptionImportPath,
					Annotations: annotations,
				},
			},
		},
	}
}

func TestMethodExceptionsMapLiteral_noThrows(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "nil", methodExceptionsMapLiteral(nil, nil, ""))
	assert.Equal(t, "nil", methodExceptionsMapLiteral(&api.Function{}, nil, ""))
	assert.Equal(t, "nil", methodExceptionsMapLiteral(&api.Function{
		Exceptions: []*api.Argument{},
	}, nil, ""))
}

func TestMethodExceptionsMapLiteral_throwsWithoutAnnotations(t *testing.T) {
	t.Parallel()
	f := &api.Function{
		Exceptions: []*api.Argument{
			exceptionArg("IntegerMismatchError", nil),
			exceptionArg("AnotherError", map[string]string{}),
		},
	}
	ns := strconv.Quote(transport.RPCCodeNotSetLiteral)
	want := expectedExceptionsMap(
		strconv.Quote("AnotherError")+`: `+ns,
		strconv.Quote("IntegerMismatchError")+`: `+ns,
	)
	assert.Equal(t, want, methodExceptionsMapLiteral(f, nil, ""))
}

func TestMethodExceptionsMapLiteral_throwsWithRPCCode(t *testing.T) {
	t.Parallel()
	f := &api.Function{
		Exceptions: []*api.Argument{
			exceptionArg("KeyDoesNotExist", map[string]string{
				_errorCodeAnnotationKey: "INVALID_ARGUMENT",
			}),
		},
	}
	want := expectedExceptionsMap(
		strconv.Quote("KeyDoesNotExist") + `: ` + strconv.Quote("INVALID_ARGUMENT"),
	)
	assert.Equal(t, want, methodExceptionsMapLiteral(f, nil, ""))
}

func TestMethodExceptionsMapLiteral_allEntriesSkipped(t *testing.T) {
	t.Parallel()
	// nil throws entry -> continue (ex nil)
	// missing Type -> continue (ex.Type nil)
	// type with no resolvable ReferenceType -> continue (ref nil)
	f := &api.Function{
		Exceptions: []*api.Argument{
			nil,
			{Type: nil},
			{Type: &api.Type{}},
			{Type: &api.Type{PointerType: &api.Type{}}}, // unwraps to empty Type
		},
	}
	assert.Equal(t, "nil", methodExceptionsMapLiteral(f, nil, ""))
}

func TestMethodExceptionsMapLiteral_mixedValidAndSkipped(t *testing.T) {
	t.Parallel()
	f := &api.Function{
		Exceptions: []*api.Argument{
			nil,
			{Type: &api.Type{}},
			exceptionArg("Kept", map[string]string{_errorCodeAnnotationKey: "NOT_FOUND"}),
		},
	}
	want := expectedExceptionsMap(
		strconv.Quote("Kept") + `: ` + strconv.Quote("NOT_FOUND"),
	)
	assert.Equal(t, want, methodExceptionsMapLiteral(f, nil, ""))
}

func TestMethodExceptionsMapLiteral_crossIncludedThriftException(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const errorThrift = `
exception InvalidArgumentError {
  1: required string message
} (
  rpc.code = "INVALID_ARGUMENT"
)
`
	const svcThrift = `
include "./error.thrift"

service Svc {
  void m() throws (1: error.InvalidArgumentError e)
}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "error.thrift"), []byte(errorThrift), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "svc.thrift"), []byte(svcThrift), 0o644))

	svcPath := filepath.Join(dir, "svc.thrift")
	mod, err := compile.Compile(svcPath)
	require.NoError(t, err)

	exc := mod.Services["Svc"].Functions["m"].ResultSpec.Exceptions[0].Type.(*compile.StructSpec)
	assert.Equal(t, "error.InvalidArgumentError", thriftExceptionMapKey(mod, exc))

	f := &api.Function{
		ThriftName: "m",
		Exceptions: []*api.Argument{
			exceptionArg("InvalidArgumentError", map[string]string{_errorCodeAnnotationKey: "INVALID_ARGUMENT"}),
		},
	}
	want := expectedExceptionsMap(
		strconv.Quote("error.InvalidArgumentError") + `: ` + strconv.Quote("INVALID_ARGUMENT"),
	)
	assert.Equal(t, want, methodExceptionsMapLiteral(f, mod, "Svc"))
}

func TestMethodExceptionsMapLiteral_duplicateThrowsLastWins(t *testing.T) {
	t.Parallel()
	ns := strconv.Quote(transport.RPCCodeNotSetLiteral)
	f := &api.Function{
		Exceptions: []*api.Argument{
			exceptionArg("DupError", map[string]string{_errorCodeAnnotationKey: "FIRST"}),
			exceptionArg("DupError", map[string]string{_errorCodeAnnotationKey: "SECOND"}),
		},
	}
	want := expectedExceptionsMap(strconv.Quote("DupError") + `: ` + strconv.Quote("SECOND"))
	assert.Equal(t, want, methodExceptionsMapLiteral(f, nil, ""))

	f2 := &api.Function{
		Exceptions: []*api.Argument{
			exceptionArg("DupError", nil),
			exceptionArg("DupError", map[string]string{_errorCodeAnnotationKey: "ONLY"}),
		},
	}
	want2 := expectedExceptionsMap(strconv.Quote("DupError") + `: ` + strconv.Quote("ONLY"))
	assert.Equal(t, want2, methodExceptionsMapLiteral(f2, nil, ""))

	f3 := &api.Function{
		Exceptions: []*api.Argument{
			exceptionArg("DupError", map[string]string{_errorCodeAnnotationKey: "ONLY"}),
			exceptionArg("DupError", nil),
		},
	}
	want3 := expectedExceptionsMap(strconv.Quote("DupError") + `: ` + ns)
	assert.Equal(t, want3, methodExceptionsMapLiteral(f3, nil, ""))
}

func TestThriftExceptionMapKey_sameThriftFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	const oneFile = `
exception LocalErr {
  1: required string message
} (
  rpc.code = "INVALID_ARGUMENT"
)

service LocalSvc {
  void ping() throws (1: LocalErr e)
}
`
	path := filepath.Join(dir, "one.thrift")
	require.NoError(t, os.WriteFile(path, []byte(oneFile), 0o644))
	mod, err := compile.Compile(path)
	require.NoError(t, err)
	exc := mod.Services["LocalSvc"].Functions["ping"].ResultSpec.Exceptions[0].Type.(*compile.StructSpec)
	assert.Equal(t, "LocalErr", thriftExceptionMapKey(mod, exc))
}
