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

func TestGetUUID(t *testing.T) {
	t.Run("happyCase", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/happy.thrift")
		require.NoError(t, err)
		annotated, err := anyAnnotatedTypes(spec)
		assert.NoError(t, err)
		assert.Equal(t, []annotatedUUIDField{
			{TypeName: "Struct", FieldName: "UserIdentifier"},
		}, annotated)
	})

	t.Run("multipleAnnotatedStructs", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/multipleAnnotatedStructs.thrift")
		require.NoError(t, err)
		annotated, err := anyAnnotatedTypes(spec)
		assert.NoError(t, err)

		// Results are sorted by (TypeName, FieldName) so the generated
		// types_yarpc_uuid.go is byte-stable across runs.
		want := []annotatedUUIDField{
			{TypeName: "GreenStruct", FieldName: "CatIdentifier"},
			{TypeName: "RedStruct", FieldName: "UserIdentifier"},
		}
		assert.Equal(t, want, annotated)
	})

	t.Run("serviceMethodArg", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/serviceArg.thrift")
		require.NoError(t, err)
		annotated, err := anyAnnotatedTypes(spec)
		assert.NoError(t, err)
		// Thrift method arg `interested` should be PascalCased to `Interested`
		// to match the field thriftrw emits on TestService_TestMethod_Args.
		assert.Equal(t, []annotatedUUIDField{
			{TypeName: "TestService_TestMethod_Args", FieldName: "Interested"},
		}, annotated)
	})

	t.Run("duplicateAnnotationsRejected", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/broken.thrift")
		require.NoError(t, err)
		_, err = anyAnnotatedTypes(spec)
		assert.Error(t, err)
	})
}

func TestGoName(t *testing.T) {
	cases := map[string]string{
		"UserIdentifier":  "UserIdentifier",
		"userIdentifier":  "UserIdentifier",
		"user_identifier": "UserIdentifier",
		"UUID":            "UUID",
		"interested":      "Interested",
	}
	for in, want := range cases {
		assert.Equal(t, want, goName(in), "goName(%q)", in)
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
}
