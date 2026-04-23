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
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/thriftrw/plugin/api"
)

// expectedExceptionsMap builds the same shape as methodExceptionsMapLiteral for assertions.
func expectedExceptionsMap(entries ...string) string {
	if len(entries) == 0 {
		return "map[string]string{}"
	}
	lines := make([]string, len(entries))
	for i, e := range entries {
		lines[i] = exceptionsMapEntryIndent + e
	}
	return "map[string]string{\n" +
		strings.Join(lines, ",\n") +
		",\n" +
		exceptionsMapCloseIndent + "}"
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
	return &api.Argument{
		Type: &api.Type{
			PointerType: &api.Type{
				ReferenceType: &api.TypeReference{
					Name:        typeName,
					Annotations: annotations,
				},
			},
		},
	}
}

func TestMethodExceptionsMapLiteral_noThrows(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "map[string]string{}", methodExceptionsMapLiteral(nil))
	assert.Equal(t, "map[string]string{}", methodExceptionsMapLiteral(&api.Function{}))
	assert.Equal(t, "map[string]string{}", methodExceptionsMapLiteral(&api.Function{
		Exceptions: []*api.Argument{},
	}))
}

func TestMethodExceptionsMapLiteral_throwsWithoutAnnotations(t *testing.T) {
	t.Parallel()
	f := &api.Function{
		Exceptions: []*api.Argument{
			exceptionArg("IntegerMismatchError", nil),
			exceptionArg("AnotherError", map[string]string{}),
		},
	}
	want := expectedExceptionsMap(
		strconv.Quote("IntegerMismatchError")+`: `+strconv.Quote("__not_set__"),
		strconv.Quote("AnotherError")+`: `+strconv.Quote("__not_set__"),
	)
	assert.Equal(t, want, methodExceptionsMapLiteral(f))
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
	assert.Equal(t, want, methodExceptionsMapLiteral(f))
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
	assert.Equal(t, "map[string]string{}", methodExceptionsMapLiteral(f))
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
	assert.Equal(t, want, methodExceptionsMapLiteral(f))
}
