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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/plugin/api"
	"go.uber.org/thriftrw/ptr"
	withservices "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/WITHSERVICES"
)

func stepNames(p *uuidPath) []string {
	if p == nil {
		return nil
	}
	out := make([]string, 0, len(p.Steps))
	for _, s := range p.Steps {
		out = append(out, s.Field.Name)
	}
	return out
}

func stepCasts(p *uuidPath) []string {
	if p == nil {
		return nil
	}
	out := make([]string, 0, len(p.Steps))
	for _, s := range p.Steps {
		switch {
		case s.CastTypeName == "":
			out = append(out, "")
		case s.CastImportPath == "":
			out = append(out, s.CastTypeName)
		default:
			out = append(out, s.CastImportPath+"."+s.CastTypeName)
		}
	}
	return out
}

func TestUUIDPathInArgs(t *testing.T) {
	t.Run("directArg", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/serviceArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"interested"}, stepNames(got[0]))
		assert.False(t, got[0].IsTypedef)
	})

	t.Run("structArgWithAnnotatedField", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/structArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"request", "UserIdentifier"}, stepNames(got[0]))
		assert.False(t, got[0].IsTypedef)
	})

	t.Run("typedefArg", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/typedefArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"identifier"}, stepNames(got[0]))
		assert.True(t, got[0].IsTypedef)
	})

	t.Run("nestedStructArgArbitraryDepth", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/nestedStruct.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"nested", "mid", "inner", "innerUUID"}, stepNames(got[0]))
		assert.False(t, got[0].IsTypedef)
	})

	t.Run("ignoredAnnotationsResolveToNoPath", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/ignoredAnnotations.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)
		assert.Nil(t, uuidPathInArgs(fn, nil))
	})

	t.Run("noAnnotation", func(t *testing.T) {
		spec, err := compile.Compile("internal/tests/atomic.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["Store"].Functions["increment"]
		require.True(t, ok)
		assert.Nil(t, uuidPathInArgs(fn, nil))
	})

	t.Run("nilFunction", func(t *testing.T) {
		assert.Nil(t, uuidPathInArgs(nil, nil))
	})

	t.Run("cyclicArgWithSiblingAnnotation", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/cycleWithSiblingAnnotation.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"arg", "id"}, stepNames(got[0]),
			"path must be arg.id (one hop), not arg.loop.id (which traverses the cycle)")
		assert.Equal(t, []string{"", ""}, stepCasts(got[0]),
			"no typedef hops along this path, so no casts are emitted")
	})

	t.Run("typedefOfStructArg", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/typedefStructArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"arg", "id"}, stepNames(got[0]))
		assert.Equal(t, []string{"Inner", ""}, stepCasts(got[0]),
			"the arg step is a typedef of Inner, so its accessor must be cast to *Inner")
		assert.False(t, got[0].IsTypedef)
	})

	t.Run("multiHopTypedefStructArg", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/multiTypedefStructArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"outer", "deeplyAliased", "id"}, stepNames(got[0]))
		assert.Equal(t, []string{"", "Inner", ""}, stepCasts(got[0]),
			"only the typedef-wrapped step needs a cast, and the cast target is the resolved underlying struct (Inner) regardless of the number of typedef hops")
		assert.False(t, got[0].IsTypedef)
	})

	t.Run("typedefOfStringLeafThroughTypedefStructHop", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/typedefStringViaTypedefStruct.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"arg", "id"}, stepNames(got[0]))
		assert.Equal(t, []string{"Inner", ""}, stepCasts(got[0]))
		assert.True(t, got[0].IsTypedef,
			"leaf typedef-of-string flag must propagate up through the typedef-of-struct hop so the template wraps the chain in string(...)")
	})

	t.Run("cycleThroughTypedefStillSurfacesSibling", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/cycleThroughTypedef.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"arg", "id"}, stepNames(got[0]),
			"both `loop` (typedef-wrapped self-ref) and `direct` (bare self-ref) must be skipped; `id` on the same level wins")
		assert.Equal(t, []string{"", ""}, stepCasts(got[0]))
	})

	t.Run("crossPackageTypedefOfStructEmitsQualifiedCast", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/crossPkgUuid.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		const (
			outerImportPath = "go.uber.org/yarpc/test/crosspkguuid"
			innerImportPath = "go.uber.org/yarpc/test/crosspkginner"
		)

		innerSS, ok := spec.Includes["crossPkgInner"].Module.Types["CrossInner"].(*compile.StructSpec)
		require.True(t, ok)
		data := &moduleTemplateData{
			Module: &api.Module{ImportPath: outerImportPath, ThriftFilePath: spec.ThriftPath},
			AllModules: map[string]*api.Module{
				filepath.Clean(spec.ThriftPath):      {ImportPath: outerImportPath, ThriftFilePath: spec.ThriftPath},
				filepath.Clean(innerSS.ThriftFile()): {ImportPath: innerImportPath, ThriftFilePath: innerSS.ThriftFile()},
			},
		}

		got := uuidPathInArgs(fn, data)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"arg", "crossUUID"}, stepNames(got[0]))
		assert.Equal(t, []string{innerImportPath + ".CrossInner", ""}, stepCasts(got[0]),
			"the cast target's import path must be the included module's package, not the local one")
	})

	t.Run("crossPackageCastFallsBackToUnqualifiedWithoutModulesMap", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/crossPkgUuid.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.Len(t, got, 1)
		assert.Equal(t, []string{"arg", "crossUUID"}, stepNames(got[0]))
		assert.Equal(t, []string{"CrossInner", ""}, stepCasts(got[0]))
	})
}

// TestMethodHasActorUUIDArg covers the server-template helper that
// gates per-method validator emission. It exercises nil inputs, an
// unknown Thrift name, an annotated method, an annotated method
// reached through a struct hop, an annotated method reached through
// a typedef-of-string arg, and a method whose args carry no
// annotation.
func TestMethodHasActorUUIDArg(t *testing.T) {
	mod, err := compile.Compile("internal/tests/WITHSERVICES.thrift")
	require.NoError(t, err)

	t.Run("nilFunction", func(t *testing.T) {
		assert.False(t, methodHasActorUUIDArg(nil, mod, "TestService"))
	})

	t.Run("unknownThriftName", func(t *testing.T) {
		fn := &api.Function{Name: "DoesNotExist", ThriftName: "doesNotExist"}
		assert.False(t, methodHasActorUUIDArg(fn, mod, "TestService"))
	})

	t.Run("annotatedArg", func(t *testing.T) {
		fn := &api.Function{Name: "TestMethod", ThriftName: "testMethod"}
		assert.True(t, methodHasActorUUIDArg(fn, mod, "TestService"))
	})

	t.Run("structArgWithAnnotatedField", func(t *testing.T) {
		fn := &api.Function{Name: "TestStructMethod", ThriftName: "testStructMethod"}
		assert.True(t, methodHasActorUUIDArg(fn, mod, "TestService"))
	})

	t.Run("annotatedTypedefArg", func(t *testing.T) {
		fn := &api.Function{Name: "TestTypedefMethod", ThriftName: "testTypedefMethod"}
		assert.True(t, methodHasActorUUIDArg(fn, mod, "TestService"))
	})

	t.Run("nestedStructArg", func(t *testing.T) {
		fn := &api.Function{Name: "TestNestedMethod", ThriftName: "testNestedMethod"}
		assert.True(t, methodHasActorUUIDArg(fn, mod, "TestService"))
	})

	t.Run("unannotatedArgs", func(t *testing.T) {
		atomicMod, err := compile.Compile("internal/tests/atomic.thrift")
		require.NoError(t, err)
		fn := &api.Function{Name: "Healthy", ThriftName: "healthy"}
		assert.False(t, methodHasActorUUIDArg(fn, atomicMod, "Store"))
	})
}

// TestServiceHasActorUUIDMethod covers the helper that gates
// service-wide emission of the actorUUIDValidator field on the wire
// handler and its initialisation in New.
func TestServiceHasActorUUIDMethod(t *testing.T) {
	withSvcMod, err := compile.Compile("internal/tests/WITHSERVICES.thrift")
	require.NoError(t, err)

	atomicMod, err := compile.Compile("internal/tests/atomic.thrift")
	require.NoError(t, err)

	t.Run("nilService", func(t *testing.T) {
		assert.False(t, serviceHasActorUUIDMethod(nil, withSvcMod))
	})

	t.Run("nilModule", func(t *testing.T) {
		svc := &api.Service{Name: "TestService", ThriftName: "TestService"}
		assert.False(t, serviceHasActorUUIDMethod(svc, nil))
	})

	t.Run("serviceWithAnnotatedMethod", func(t *testing.T) {
		svc := &api.Service{Name: "TestService", ThriftName: "TestService"}
		assert.True(t, serviceHasActorUUIDMethod(svc, withSvcMod))
	})

	t.Run("serviceWithoutAnnotatedMethods", func(t *testing.T) {
		svc := &api.Service{Name: "Store", ThriftName: "Store"}
		assert.False(t, serviceHasActorUUIDMethod(svc, atomicMod))
	})

	t.Run("unknownService", func(t *testing.T) {
		svc := &api.Service{Name: "NotThere", ThriftName: "NotThere"}
		assert.False(t, serviceHasActorUUIDMethod(svc, withSvcMod))
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

// TestGetGeneratedUUID exercises the ActorUUID() accessors thriftrw
// emits on the WITHSERVICES package, end-to-end. The plugin only
// generates these accessors on Service_Method_Args structs (the
// server template's only call site), so every subtest constructs a
// method-args value and asserts the GetXxx() chain the generator
// wired up surfaces the expected UUID, including the nil-safe case
// for missing intermediate struct hops.
func TestGetGeneratedUUID(t *testing.T) {
	t.Run("arg field UUID", func(t *testing.T) {
		st := withservices.TestService_TestMethod_Args{
			Interested: ptr.String("my-arg-uuid"),
		}
		assert.Equal(t, []string{"my-arg-uuid"}, st.ActorUUID())
	})
	t.Run("struct arg with annotated field", func(t *testing.T) {
		st := withservices.TestService_TestStructMethod_Args{
			Request: &withservices.Struct{
				UserIdentifier: ptr.String("my-struct-uuid"),
			},
		}
		assert.Equal(t, []string{"my-struct-uuid"}, st.ActorUUID())
	})
	t.Run("struct arg with nil request", func(t *testing.T) {
		var st withservices.TestService_TestStructMethod_Args
		assert.Equal(t, []string{""}, st.ActorUUID())
	})
	t.Run("typedef-of-string arg field UUID", func(t *testing.T) {
		id := withservices.ActorIdentifier("my-typedef-uuid")
		st := withservices.TestService_TestTypedefMethod_Args{
			Identifier: &id,
		}
		assert.Equal(t, []string{"my-typedef-uuid"}, st.ActorUUID())
	})
	t.Run("nested struct arg walks full chain", func(t *testing.T) {
		st := withservices.TestService_TestNestedMethod_Args{
			Nested: &withservices.OuterLevel{
				Mid: &withservices.MidLevel{
					Inner: &withservices.InnerLevel{
						InnerUUID: ptr.String("my-nested-uuid"),
					},
				},
			},
		}
		assert.Equal(t, []string{"my-nested-uuid"}, st.ActorUUID())
	})
	t.Run("nested struct arg with nil mid is nil-safe", func(t *testing.T) {
		st := withservices.TestService_TestNestedMethod_Args{
			Nested: &withservices.OuterLevel{},
		}
		assert.Equal(t, []string{""}, st.ActorUUID())
	})
	t.Run("typedef-of-struct arg casts through underlying struct", func(t *testing.T) {
		val := withservices.AliasedInner{InnerUUID: ptr.String("typedef-uuid")}
		st := withservices.TestService_TestTypedefStructMethod_Args{TopLevel: &val}
		assert.Equal(t, []string{"typedef-uuid"}, st.ActorUUID())
	})
	t.Run("typedef-of-struct arg with nil receiver is nil-safe", func(t *testing.T) {
		var st withservices.TestService_TestTypedefStructMethod_Args
		assert.Equal(t, []string{""}, st.ActorUUID())
	})
	t.Run("nested typedef-of-struct chain casts at the typedef hop only", func(t *testing.T) {
		val := withservices.AliasedInner{InnerUUID: ptr.String("nested-typedef-uuid")}
		st := withservices.TestService_TestNestedTypedefStructMethod_Args{
			Outer: &withservices.OuterWithAlias{Inner: &val},
		}
		assert.Equal(t, []string{"nested-typedef-uuid"}, st.ActorUUID())
	})
	t.Run("nested typedef-of-struct chain with nil typedef hop is nil-safe", func(t *testing.T) {
		st := withservices.TestService_TestNestedTypedefStructMethod_Args{
			Outer: &withservices.OuterWithAlias{},
		}
		assert.Equal(t, []string{""}, st.ActorUUID())
	})
	t.Run("two-hop typedef arg uses a single direct cast", func(t *testing.T) {
		val := withservices.DoubleAliasedInner(withservices.AliasedInner{
			InnerUUID: ptr.String("two-hop-uuid"),
		})
		st := withservices.TestService_TestDoubleTypedefStructMethod_Args{Arg: &val}
		assert.Equal(t, []string{"two-hop-uuid"}, st.ActorUUID())
	})
	t.Run("two-hop typedef arg with nil receiver is nil-safe", func(t *testing.T) {
		var st withservices.TestService_TestDoubleTypedefStructMethod_Args
		assert.Equal(t, []string{""}, st.ActorUUID())
	})
	t.Run("two annotated flat args yield one slice entry each", func(t *testing.T) {
		st := withservices.TestService_TestMultiAnnotatedArgsMethod_Args{
			FirstActor:  ptr.String("first-uuid"),
			SecondActor: ptr.String("second-uuid"),
		}
		assert.Equal(t, []string{"first-uuid", "second-uuid"}, st.ActorUUID(),
			"each annotated arg contributes its own entry, in declaration order")
	})
	t.Run("same leaf reached via two sibling fields yields two entries", func(t *testing.T) {
		st := withservices.TestService_TestTwoStructPathsMethod_Args{
			Pair: &withservices.PairedStructs{
				Primary:   &withservices.Struct{UserIdentifier: ptr.String("primary-uuid")},
				Secondary: &withservices.Struct{UserIdentifier: ptr.String("secondary-uuid")},
			},
		}
		assert.Equal(t, []string{"primary-uuid", "secondary-uuid"}, st.ActorUUID(),
			"primary and secondary are distinct runtime fields, so both routes to UserIdentifier are emitted")
	})
	t.Run("two annotated args mixing flat and struct-hop shapes", func(t *testing.T) {
		st := withservices.TestService_TestMixedAnnotatedMethod_Args{
			DirectActor: ptr.String("direct-uuid"),
			Request:     &withservices.Struct{UserIdentifier: ptr.String("struct-uuid")},
		}
		assert.Equal(t, []string{"direct-uuid", "struct-uuid"}, st.ActorUUID())
	})
	t.Run("multi-annotation slice is nil-safe per entry", func(t *testing.T) {
		st := withservices.TestService_TestTwoStructPathsMethod_Args{
			Pair: &withservices.PairedStructs{
				Primary: &withservices.Struct{UserIdentifier: ptr.String("only-primary")},
			},
		}
		assert.Equal(t, []string{"only-primary", ""}, st.ActorUUID(),
			"a missing sibling surfaces as the empty string but keeps its slice position")
	})
}
