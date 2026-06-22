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
		require.NotNil(t, got)
		assert.Equal(t, []string{"arg", "crossUUID"}, stepNames(got))
		assert.Equal(t, []string{innerImportPath + ".CrossInner", ""}, stepCasts(got),
			"the cast target's import path must be the included module's package, not the local one")
	})

	t.Run("crossPackageCastFallsBackToUnqualifiedWithoutModulesMap", func(t *testing.T) {
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

// TestValidateUUIDArgs covers the per-function validator that the
// plugin runs upfront in Generate before any code is emitted: it
// must accept a missing function, a function with no annotations,
// and a function with exactly one reachable annotation, and reject
// a function whose args reach more than one annotated leaf — even
// when one of those leaves sits on a struct that participates in a
// cycle.
func TestValidateUUIDArgs(t *testing.T) {
	t.Run("nilFunction", func(t *testing.T) {
		assert.NoError(t, validateUUIDArgs(nil))
	})

	t.Run("noAnnotationIsOK", func(t *testing.T) {
		spec, err := compile.Compile("internal/tests/atomic.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["Store"].Functions["increment"]
		require.True(t, ok)
		assert.NoError(t, validateUUIDArgs(fn))
	})

	t.Run("singleAnnotationIsOK", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/serviceArg.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)
		assert.NoError(t, validateUUIDArgs(fn))
	})

	t.Run("multiplePathsToSameLeafErrors", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/twoPathsToSameLeaf.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		err = validateUUIDArgs(fn)
		require.Error(t, err)
		for _, want := range []string{"outer.first.id", "outer.second.id"} {
			assert.Contains(t, err.Error(), want,
				"both ambiguous paths must appear in the diagnostic so the user can pick which one to keep")
		}
	})

	t.Run("multipleAnnotatedArgsErrors", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/multipleAnnotatedArgs.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		err = validateUUIDArgs(fn)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth.actor_uuid")
		assert.Contains(t, err.Error(), `"testMethod"`,
			"the offending method name must appear in the diagnostic so it points at the .thrift declaration")
		for _, want := range []string{"firstUUID", "second.id", "thirdUUID"} {
			assert.Contains(t, err.Error(), want,
				"error message must list every reachable annotated path")
		}
	})

	t.Run("multipleAnnotationsAcrossCycleErrors", func(t *testing.T) {
		spec, err := compile.Compile("internal/uuid_test/multipleAnnotationsAcrossCycle.thrift")
		require.NoError(t, err)
		fn, ok := spec.Services["TestService"].Functions["testMethod"]
		require.True(t, ok)

		err = validateUUIDArgs(fn)
		require.Error(t, err)
		for _, want := range []string{"arg.idInsideCycle", "idOutside"} {
			assert.Contains(t, err.Error(), want,
				"annotation reachable inside a cycle must still be reported alongside its non-cyclic sibling")
		}
	})
}

// TestValidateUUIDArgsInModule covers the module-level wrapper used
// by the plugin's Generate entry point: it walks every service and
// every function in the compiled module and surfaces the first
// multi-annotation error in deterministic order (services and
// methods are scanned in alphabetical Thrift-name order, which
// matters because ThriftRW exposes them as Go maps with randomised
// iteration).
func TestValidateUUIDArgsInModule(t *testing.T) {
	t.Run("nilModule", func(t *testing.T) {
		assert.NoError(t, validateUUIDArgsInModule(nil))
	})

	t.Run("cleanModuleWithNoAnnotations", func(t *testing.T) {
		mod, err := compile.Compile("internal/tests/atomic.thrift")
		require.NoError(t, err)
		assert.NoError(t, validateUUIDArgsInModule(mod))
	})

	t.Run("cleanModuleWithSingleAnnotationsPerMethod", func(t *testing.T) {
		// WITHSERVICES.thrift has many methods, each with at most
		// one reachable annotation. The module-level walker must
		// accept all of them as a group.
		mod, err := compile.Compile("internal/tests/WITHSERVICES.thrift")
		require.NoError(t, err)
		assert.NoError(t, validateUUIDArgsInModule(mod))
	})

	t.Run("multipleAnnotatedMethodErrorsWithServiceAndMethodNamed", func(t *testing.T) {
		mod, err := compile.Compile("internal/uuid_test/multipleAnnotatedArgs.thrift")
		require.NoError(t, err)
		err = validateUUIDArgsInModule(mod)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `"TestService"`,
			"the wrapper must name the offending service so the user can locate it in a multi-service file")
		assert.Contains(t, err.Error(), `"testMethod"`,
			"the wrapped per-function error must still name the method")
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
		val := withservices.AliasedInner{InnerUUID: ptr.String("typedef-uuid")}
		st := withservices.TestService_TestTypedefStructMethod_Args{TopLevel: &val}
		assert.Equal(t, "typedef-uuid", st.ActorUUID())
	})
	t.Run("typedef-of-struct arg with nil receiver is nil-safe", func(t *testing.T) {
		var st withservices.TestService_TestTypedefStructMethod_Args
		assert.Equal(t, "", st.ActorUUID())
	})
	t.Run("nested typedef-of-struct chain casts at the typedef hop only", func(t *testing.T) {
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
