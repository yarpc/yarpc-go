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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/ptr"
	withservices "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/WITHSERVICES"
)

// TestValidateUUIDAnnotations covers the pre-flight pass that rejects
// multiple annotations per container. The detailed per-type/per-method
// lookup that the template performs at execution time is covered by
// TestUUIDFieldOf, TestUUIDFieldInArgs and end-to-end by TestCodeIsUpToDate.
func TestValidateUUIDAnnotations(t *testing.T) {
	t.Run("happyCase", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/happy.thrift")
		require.NoError(t, err)
		assert.NoError(t, validateUUIDAnnotations(spec))
	})

	t.Run("multipleAnnotatedStructs", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/multipleAnnotatedStructs.thrift")
		require.NoError(t, err)
		assert.NoError(t, validateUUIDAnnotations(spec))
	})

	t.Run("serviceMethodArg", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/serviceArg.thrift")
		require.NoError(t, err)
		assert.NoError(t, validateUUIDAnnotations(spec))
	})

	t.Run("duplicateAnnotationsRejected", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/broken.thrift")
		require.NoError(t, err)
		assert.Error(t, validateUUIDAnnotations(spec))
	})

	t.Run("twoAnnotatedArgsRejected", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/brokenTwoAnnotatedArgs.thrift")
		require.NoError(t, err)
		assert.Error(t, validateUUIDAnnotations(spec))
	})

	t.Run("primitiveAndStructArgRejected", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/brokenPrimitiveAndStructArg.thrift")
		require.NoError(t, err)
		assert.Error(t, validateUUIDAnnotations(spec))
	})

	t.Run("twoStructArgsRejected", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/brokenTwoStructArgs.thrift")
		require.NoError(t, err)
		assert.Error(t, validateUUIDAnnotations(spec))
	})
}

// TestUUIDFieldOf covers the template helper that picks the annotated field
// on a Thrift struct.
func TestUUIDFieldOf(t *testing.T) {
	t.Run("structWithAnnotatedField", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/happy.thrift")
		require.NoError(t, err)
		ss, ok := spec.Types["Struct"].(*compile.StructSpec)
		require.True(t, ok)

		got := uuidFieldOf(ss)
		require.NotNil(t, got)
		assert.Equal(t, "UserIdentifier", got.Field.Name)
		assert.False(t, got.IsTypedef)
	})

	t.Run("multipleAnnotatedStructsResolveIndependently", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/multipleAnnotatedStructs.thrift")
		require.NoError(t, err)

		red, ok := spec.Types["RedStruct"].(*compile.StructSpec)
		require.True(t, ok)
		gotRed := uuidFieldOf(red)
		require.NotNil(t, gotRed)
		assert.Equal(t, "UserIdentifier", gotRed.Field.Name)

		green, ok := spec.Types["GreenStruct"].(*compile.StructSpec)
		require.True(t, ok)
		gotGreen := uuidFieldOf(green)
		require.NotNil(t, gotGreen)
		assert.Equal(t, "CatIdentifier", gotGreen.Field.Name)
	})

	t.Run("nonStructTypeSpec", func(t *testing.T) {
		assert.Nil(t, uuidFieldOf(&compile.StringSpec{}))
	})
}

// TestUUIDFieldInArgs covers the template helper that picks the annotated
// field within a method's argument list.
func TestUUIDFieldInArgs(t *testing.T) {
	t.Run("directArg", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/serviceArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidFieldInArgs(fn)
		require.NotNil(t, got)
		// Field name is the raw Thrift name; goCase runs in the
		// template and produces the PascalCased "Interested".
		assert.Equal(t, "interested", got.Field.Name)
		assert.False(t, got.IsStruct)
		assert.False(t, got.IsTypedef)
	})

	t.Run("structArgWithAnnotatedField", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/structArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidFieldInArgs(fn)
		require.NotNil(t, got)
		assert.Equal(t, "request", got.Field.Name)
		assert.True(t, got.IsStruct)
		assert.False(t, got.IsTypedef)
	})

	t.Run("typedefArg", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/typedefArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidFieldInArgs(fn)
		require.NotNil(t, got)
		assert.Equal(t, "identifier", got.Field.Name)
		assert.False(t, got.IsStruct)
		assert.True(t, got.IsTypedef)
	})

	t.Run("noAnnotation", func(t *testing.T) {
		spec, err := compile.Compile("internal/tests/atomic.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["Store"].Functions["increment"]
		require.True(t, ok)
		assert.Nil(t, uuidFieldInArgs(fn))
	})
}

func TestGoCase(t *testing.T) {
	cases := map[string]string{
		"UserIdentifier":  "UserIdentifier",
		"userIdentifier":  "UserIdentifier",
		"user_identifier": "UserIdentifier",
		"UUID":            "UUID",
		"interested":      "Interested",
	}
	for in, want := range cases {
		assert.Equal(t, want, goCase(in), "goCase(%q)", in)
	}
}

func TestGetGeneratedUUID(t *testing.T) {
	t.Run("optional field UUID", func(t *testing.T) {
		st := withservices.Struct{
			Baz:            ptr.String("test"),
			UserIdentifier: ptr.String("my-uuid"),
		}
		assert.Equal(t, "my-uuid", st.ActorUUID())
	})
	t.Run("required field UUID", func(t *testing.T) {
		st := withservices.StructRequiredUUID{
			Baz:            ptr.String("test"),
			UserIdentifier: "my-required-uuid",
		}
		assert.Equal(t, "my-required-uuid", st.ActorUUID())
	})
	t.Run("arg field UUID", func(t *testing.T) {
		st := withservices.TestService_TestMethod_Args{
			Interested: ptr.String("my-arg-uuid"),
		}
		assert.Equal(t, "my-arg-uuid", st.ActorUUID())
	})
	t.Run("struct arg with annotated field", func(t *testing.T) {
		st := withservices.TestService_TestStructMethod_Args{
			Request: &withservices.Struct{
				UserIdentifier: ptr.String("my-struct-uuid"),
			},
		}
		assert.Equal(t, "my-struct-uuid", st.ActorUUID())
	})
	t.Run("struct arg with nil request", func(t *testing.T) {
		var st withservices.TestService_TestStructMethod_Args
		assert.Equal(t, "", st.ActorUUID())
	})
	t.Run("typedef-of-string arg field UUID", func(t *testing.T) {
		id := withservices.ActorIdentifier("my-typedef-uuid")
		st := withservices.TestService_TestTypedefMethod_Args{
			Identifier: &id,
		}
		assert.Equal(t, "my-typedef-uuid", st.ActorUUID())
	})
}
