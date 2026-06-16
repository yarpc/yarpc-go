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
		require.NotNil(t, got)
		assert.Equal(t, []string{"interested"}, stepNames(got))
		assert.False(t, got.IsTypedef)
	})

	t.Run("structArgWithAnnotatedField", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/structArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.NotNil(t, got)
		assert.Equal(t, []string{"request", "UserIdentifier"}, stepNames(got))
		assert.False(t, got.IsTypedef)
	})

	t.Run("typedefArg", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/typedefArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.NotNil(t, got)
		assert.Equal(t, []string{"identifier"}, stepNames(got))
		assert.True(t, got.IsTypedef)
	})

	t.Run("nestedStructArgArbitraryDepth", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/nestedStruct.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.NotNil(t, got)
		assert.Equal(t, []string{"nested", "mid", "inner", "innerUUID"}, stepNames(got))
		assert.False(t, got.IsTypedef)
	})

	t.Run("multipleAnnotatedArgsTakeFirst", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/multipleAnnotatedArgs.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.NotNil(t, got)
		assert.Equal(t, []string{"firstUUID"}, stepNames(got),
			"first annotated arg by position wins; later annotated args and "+
				"annotations reachable through other args are silently ignored")
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
		require.NotNil(t, got)
		assert.Equal(t, []string{"arg", "id"}, stepNames(got),
			"path must be arg.id (one hop), not arg.loop.id (which traverses the cycle)")
		assert.Equal(t, []string{"", ""}, stepCasts(got),
			"no typedef hops along this path, so no casts are emitted")
	})

	t.Run("typedefOfStructArg", func(t *testing.T) {
		// AliasedInner is `typedef Inner AliasedInner`. The walker
		// must peel the typedef to descend into Inner's fields, and
		// the step on AliasedInner must be flagged with a same-
		// package cast through *Inner so the chain compiles
		// (thriftrw emits GetID() only on *Inner).
		spec, err := compile.Compile("internal/uuid_test/typedefStructArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.NotNil(t, got)
		assert.Equal(t, []string{"arg", "id"}, stepNames(got))
		assert.Equal(t, []string{"Inner", ""}, stepCasts(got),
			"the arg step is a typedef of Inner, so its accessor must be cast to *Inner")
		assert.False(t, got.IsTypedef)
	})

	t.Run("multiHopTypedefStructArg", func(t *testing.T) {
		// DoubleAliasedInner is a two-hop typedef chain to Inner.
		// Go allows converting any pointer type to *Inner as long
		// as the underlying struct is the same, so a single cast to
		// *Inner is sufficient regardless of how many typedefs sit
		// on the chain.
		spec, err := compile.Compile("internal/uuid_test/multiTypedefStructArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.NotNil(t, got)
		assert.Equal(t, []string{"outer", "deeplyAliased", "id"}, stepNames(got))
		assert.Equal(t, []string{"", "Inner", ""}, stepCasts(got),
			"only the typedef-wrapped step needs a cast, and the cast target is the resolved underlying struct (Inner) regardless of the number of typedef hops")
		assert.False(t, got.IsTypedef)
	})

	t.Run("typedefOfStringLeafThroughTypedefStructHop", func(t *testing.T) {
		// AliasedInner wraps Inner; Inner.id is a typedef-of-string
		// (ActorIdentifier) carrying the annotation. The walker
		// must produce both transformations: a cast on the
		// typedef-of-struct step, and IsTypedef=true so the
		// template wraps the chain in string(...).
		spec, err := compile.Compile("internal/uuid_test/typedefStringViaTypedefStruct.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.NotNil(t, got)
		assert.Equal(t, []string{"arg", "id"}, stepNames(got))
		assert.Equal(t, []string{"Inner", ""}, stepCasts(got))
		assert.True(t, got.IsTypedef,
			"leaf typedef-of-string flag must propagate up through the typedef-of-struct hop so the template wraps the chain in string(...)")
	})

	t.Run("cycleThroughTypedefStillSurfacesSibling", func(t *testing.T) {
		// CycleNode has a typedef-wrapped self-reference (`loop` of
		// type AliasedNode = CycleNode), a direct self-reference
		// (`direct`), and an annotated sibling (`id`). The walker
		// keys its visited set on the resolved struct, so both
		// cyclic fields are skipped regardless of typedef wrapping
		// and `id` is reached through arg directly.
		spec, err := compile.Compile("internal/uuid_test/cycleThroughTypedef.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.NotNil(t, got)
		assert.Equal(t, []string{"arg", "id"}, stepNames(got),
			"both `loop` (typedef-wrapped self-ref) and `direct` (bare self-ref) must be skipped; `id` on the same level wins")
		assert.Equal(t, []string{"", ""}, stepCasts(got))
	})

	t.Run("crossPackageTypedefOfStructEmitsQualifiedCast", func(t *testing.T) {
		// AliasedCrossInner is `typedef crossPkgInner.CrossInner
		// AliasedCrossInner` declared in crossPkgUuid.thrift; the
		// resolved struct lives in a different .thrift file (and
		// hence a different generated Go package). With a properly
		// populated AllModules map, the walker must record both the
		// cast type name (CrossInner) and the import path of the
		// package that owns it.
		spec, err := compile.Compile("internal/uuid_test/crossPkgUuid.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		const (
			outerImportPath = "go.uber.org/yarpc/test/crosspkguuid"
			innerImportPath = "go.uber.org/yarpc/test/crosspkginner"
		)
		// Synthesize an AllModules table keyed on the cleaned
		// thrift file paths so castImportPath can resolve
		// CrossInner's owning package even though the test does
		// not run the full plugin.
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
		require.NotNil(t, got)
		assert.Equal(t, []string{"arg", "crossUUID"}, stepNames(got))
		assert.Equal(t, []string{innerImportPath + ".CrossInner", ""}, stepCasts(got),
			"the cast target's import path must be the included module's package, not the local one")
	})

	t.Run("crossPackageCastFallsBackToUnqualifiedWithoutModulesMap", func(t *testing.T) {
		// Same fixture as above, but the test passes nil for the
		// data argument (mirroring how methodHasActorUUIDArg and
		// the other gating helpers call uuidPathInArgs without
		// import-resolution context). The walker must still
		// descend correctly and emit a same-package cast (no
		// qualifier), since it has no way to know the cast target
		// lives elsewhere.
		spec, err := compile.Compile("internal/uuid_test/crossPkgUuid.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		got := uuidPathInArgs(fn, nil)
		require.NotNil(t, got)
		assert.Equal(t, []string{"arg", "crossUUID"}, stepNames(got))
		assert.Equal(t, []string{"CrossInner", ""}, stepCasts(got))
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
		assert.Equal(t, "my-nested-uuid", st.ActorUUID())
	})
	t.Run("nested struct arg with nil mid is nil-safe", func(t *testing.T) {
		st := withservices.TestService_TestNestedMethod_Args{
			Nested: &withservices.OuterLevel{},
		}
		assert.Equal(t, "", st.ActorUUID())
	})
	t.Run("typedef-of-struct arg casts through underlying struct", func(t *testing.T) {
		// Verifies the (*Inner)(...) cast emitted in the chain:
		// thriftrw's GetTopLevel returns *AliasedInner, which has
		// no GetInnerUUID; the cast is what makes the chain
		// compile and surface the UUID.
		val := withservices.AliasedInner{InnerUUID: ptr.String("typedef-uuid")}
		st := withservices.TestService_TestTypedefStructMethod_Args{TopLevel: &val}
		assert.Equal(t, "typedef-uuid", st.ActorUUID())
	})
	t.Run("typedef-of-struct arg with nil receiver is nil-safe", func(t *testing.T) {
		var st withservices.TestService_TestTypedefStructMethod_Args
		assert.Equal(t, "", st.ActorUUID())
	})
	t.Run("nested typedef-of-struct chain casts at the typedef hop only", func(t *testing.T) {
		// OuterWithAlias has both AliasedInner and DoubleAliasedInner
		// fields; the walker picks the first reachable annotation
		// (AliasedInner) and the cast wraps that one step. The
		// outer struct itself is bare (not a typedef), so its hop
		// emits no cast.
		val := withservices.AliasedInner{InnerUUID: ptr.String("nested-typedef-uuid")}
		st := withservices.TestService_TestNestedTypedefStructMethod_Args{
			Outer: &withservices.OuterWithAlias{Inner: &val},
		}
		assert.Equal(t, "nested-typedef-uuid", st.ActorUUID())
	})
	t.Run("nested typedef-of-struct chain with nil typedef hop is nil-safe", func(t *testing.T) {
		st := withservices.TestService_TestNestedTypedefStructMethod_Args{
			Outer: &withservices.OuterWithAlias{},
		}
		assert.Equal(t, "", st.ActorUUID())
	})
	t.Run("two-hop typedef arg uses a single direct cast", func(t *testing.T) {
		// The arg's static type is DoubleAliasedInner, a two-hop
		// typedef chain to Inner. The generated chain is
		// (*Inner)(t.GetArg()).GetInnerUUID() — a single
		// pointer-conversion that jumps over the AliasedInner
		// intermediate. This proves multi-hop typedef chains do
		// not need per-hop information at codegen time: the cast
		// target is always the resolved underlying struct.
		val := withservices.DoubleAliasedInner(withservices.AliasedInner{
			InnerUUID: ptr.String("two-hop-uuid"),
		})
		st := withservices.TestService_TestDoubleTypedefStructMethod_Args{Arg: &val}
		assert.Equal(t, "two-hop-uuid", st.ActorUUID())
	})
	t.Run("two-hop typedef arg with nil receiver is nil-safe", func(t *testing.T) {
		var st withservices.TestService_TestDoubleTypedefStructMethod_Args
		assert.Equal(t, "", st.ActorUUID())
	})
}
