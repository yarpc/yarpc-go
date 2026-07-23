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

package lib

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/yarpc/internal/protoplugin"
)

// _testActorUUIDFieldNumber is a deliberately uncommon field number used in
// tests. It is registered globally with gogo/proto via init() so that
// proto.SetExtension can populate FieldOptions in a way that
// proto.Marshal/Unmarshal will round-trip into raw extension bytes - which is
// what protoc produces in real generation.
const _testActorUUIDFieldNumber = 99999

var _testActorUUIDExt = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FieldOptions)(nil),
	ExtensionType: (*bool)(nil),
	Field:         _testActorUUIDFieldNumber,
	Name:          _ActorUUIDFQN,
	Tag:           "varint,99999,opt,name=actor_uuid",
	Filename:      "uber/auth/annotations/options_test.proto",
}

func init() {
	proto.RegisterExtension(_testActorUUIDExt)
}

func TestFindActorUUIDFieldNumber(t *testing.T) {
	t.Run("present_in_dependency", func(t *testing.T) {
		optionsFile := newOptionsFile()
		targetFile := &protoplugin.File{
			FileDescriptorProto: &descriptor.FileDescriptorProto{
				Name:    proto.String("svc/foo.proto"),
				Package: proto.String("svc"),
			},
			TransitiveDependencies: []*protoplugin.File{optionsFile},
		}
		assert.Equal(t, int32(_testActorUUIDFieldNumber), findActorUUIDFieldNumber(targetFile))
	})

	t.Run("absent", func(t *testing.T) {
		targetFile := &protoplugin.File{
			FileDescriptorProto: &descriptor.FileDescriptorProto{
				Name:    proto.String("svc/foo.proto"),
				Package: proto.String("svc"),
			},
		}
		assert.Equal(t, int32(0), findActorUUIDFieldNumber(targetFile))
	})

	t.Run("present_in_target_file_itself", func(t *testing.T) {
		// Edge case: target .proto itself defines the extension.
		targetFile := newOptionsFile()
		assert.Equal(t, int32(_testActorUUIDFieldNumber), findActorUUIDFieldNumber(targetFile))
	})

	t.Run("ignored_when_extending_other_options", func(t *testing.T) {
		// Same FQN, but extending MessageOptions instead of FieldOptions:
		// must be ignored.
		fileWithWrongExtendee := &protoplugin.File{
			FileDescriptorProto: &descriptor.FileDescriptorProto{
				Name:    proto.String("uber/auth/annotations/options.proto"),
				Package: proto.String("uber.auth.annotations"),
				Extension: []*descriptor.FieldDescriptorProto{{
					Name:     proto.String("actor_uuid"),
					Number:   proto.Int32(_testActorUUIDFieldNumber),
					Type:     descriptor.FieldDescriptorProto_TYPE_BOOL.Enum(),
					Label:    descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Extendee: proto.String(".google.protobuf.MessageOptions"),
				}},
			},
		}
		target := &protoplugin.File{
			FileDescriptorProto: &descriptor.FileDescriptorProto{
				Name: proto.String("svc/foo.proto"),
			},
			TransitiveDependencies: []*protoplugin.File{fileWithWrongExtendee},
		}
		assert.Equal(t, int32(0), findActorUUIDFieldNumber(target))
	})
}

func TestHasActorUUID(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		opts := withActorUUIDOption(t, true)
		assert.True(t, hasActorUUID(opts, _testActorUUIDFieldNumber))
	})

	t.Run("explicit_false", func(t *testing.T) {
		opts := withActorUUIDOption(t, false)
		assert.False(t, hasActorUUID(opts, _testActorUUIDFieldNumber))
	})

	t.Run("not_set", func(t *testing.T) {
		assert.False(t, hasActorUUID(&descriptor.FieldOptions{}, _testActorUUIDFieldNumber))
	})

	t.Run("nil_options", func(t *testing.T) {
		assert.False(t, hasActorUUID(nil, _testActorUUIDFieldNumber))
	})

	t.Run("zero_field_number", func(t *testing.T) {
		opts := withActorUUIDOption(t, true)
		assert.False(t, hasActorUUID(opts, 0))
	})
}

// stepNames returns the root-to-leaf field names of the path.
// Mirrors the helper in encoding/thrift/thriftrw-plugin-yarpc/uuid_test.go.
func stepNames(p *uuidPath) []string {
	if p == nil {
		return nil
	}
	out := make([]string, 0, len(p.Steps))
	for _, s := range p.Steps {
		out = append(out, s.Field.GetName())
	}
	return out
}

// allStepNames flattens the per-path step names of a walk result so a
// multi-path expectation can be asserted in one comparison.
func allStepNames(paths []*uuidPath) [][]string {
	out := make([][]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, stepNames(p))
	}
	return out
}

// stepKinds returns the per-step traversal kinds of the path, used to
// assert that container hops/leaves are classified correctly.
func stepKinds(p *uuidPath) []uuidStepKind {
	if p == nil {
		return nil
	}
	out := make([]uuidStepKind, 0, len(p.Steps))
	for _, s := range p.Steps {
		out = append(out, s.Kind)
	}
	return out
}

// TestWalkForUUID exercises the depth-first path walker in isolation,
// using synthetic in-memory descriptors so each subtest is independent
// of the runtime gogo proto registry. Mirrors the structure of
// encoding/thrift/thriftrw-plugin-yarpc/uuid_test.go's TestUUIDPathInArgs.
func TestWalkForUUID(t *testing.T) {
	t.Run("directLeafField", func(t *testing.T) {
		// Single-step case: the request itself has the annotated field.
		req := newMessage(t, "Req", stringField(t, "actor", true))
		paths := walkPathOf(t, req)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"actor"}, stepNames(paths[0]))
	})

	t.Run("nestedStructArbitraryDepth", func(t *testing.T) {
		// Three-hop chain Req -> outer -> mid -> inner.uuid.
		inner := newMessage(t, "Inner", stringField(t, "uuid", true))
		mid := newMessage(t, "Mid", messageField(t, "inner", inner))
		outer := newMessage(t, "Outer", messageField(t, "mid", mid))
		req := newMessage(t, "Req", messageField(t, "outer", outer))
		paths := walkPathOf(t, req, inner, mid, outer)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"outer", "mid", "inner", "uuid"}, stepNames(paths[0]))
	})

	t.Run("multipleAnnotatedFieldsAllCollected", func(t *testing.T) {
		// Two annotated fields on the same message; the walker collects
		// both, in declaration order. Reaching more than one annotation
		// is not an error.
		req := newMessage(t, "Req",
			stringField(t, "first_actor", true),
			stringField(t, "second_actor", true),
		)
		paths := walkPathOf(t, req)
		assert.Equal(t, [][]string{{"first_actor"}, {"second_actor"}}, allStepNames(paths))
	})

	t.Run("twoSiblingMessagePathsBothCollected", func(t *testing.T) {
		// Both `primary` and `secondary` point at the same Struct type,
		// which has an annotated `uuid`. Because the visited set is
		// unwound after each branch, both sibling routes to the shared
		// type are surfaced as separate paths.
		shared := newMessage(t, "Struct", stringField(t, "uuid", true))
		req := newMessage(t, "Req",
			messageField(t, "primary", shared),
			messageField(t, "secondary", shared),
		)
		paths := walkPathOf(t, req, shared)
		assert.Equal(t, [][]string{
			{"primary", "uuid"},
			{"secondary", "uuid"},
		}, allStepNames(paths),
			"primary and secondary are distinct fields, so both routes to the shared type are emitted")
	})

	t.Run("annotationsOnNonStringIgnored", func(t *testing.T) {
		// Annotation on an int64 field is not a valid leaf; the walker
		// must ignore it and surface no path. A repeated non-string
		// scalar is likewise ignored: only string-valued containers are
		// collected.
		req := newMessage(t, "Req",
			int64Field(t, "timestamp", true),
			repeatedInt64Field(t, "timestamps", true),
		)
		assert.Empty(t, walkPathOf(t, req))
	})

	t.Run("repeatedStringLeafCollected", func(t *testing.T) {
		// A repeated string carrying the annotation is now a valid leaf:
		// every element is collected at runtime.
		req := newMessage(t, "Req", repeatedStringField(t, "actors", true))
		paths := walkPathOf(t, req)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"actors"}, stepNames(paths[0]))
		assert.Equal(t, []uuidStepKind{stepRepeatedStringLeaf}, stepKinds(paths[0]))
	})

	t.Run("mapStringLeafCollected", func(t *testing.T) {
		// A map<string, string> carrying the annotation is a valid leaf:
		// every value is collected.
		entry := mapEntryMessage(t, "ActorsByRoleEntry", stringValueField(t))
		req := newMessage(t, "Req", mapField(t, "actors_by_role", entry, true))
		paths := walkPathOf(t, req, entry)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"actors_by_role"}, stepNames(paths[0]))
		assert.Equal(t, []uuidStepKind{stepMapStringLeaf}, stepKinds(paths[0]))
	})

	t.Run("mapToNonStringValueIgnored", func(t *testing.T) {
		// A map<string, int64> annotation is ignored: the value is not a
		// string, so there is no leaf to collect.
		entry := mapEntryMessage(t, "CountsEntry", int64ValueField(t))
		req := newMessage(t, "Req", mapField(t, "counts", entry, true))
		assert.Empty(t, walkPathOf(t, req, entry))
	})

	t.Run("repeatedMessageHopDescended", func(t *testing.T) {
		// A repeated message hop is descended: the walker ranges into the
		// element type and surfaces its annotated leaf.
		item := newMessage(t, "Item", stringField(t, "uuid", true))
		req := newMessage(t, "Req", repeatedMessageField(t, "items", item))
		paths := walkPathOf(t, req, item)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"items", "uuid"}, stepNames(paths[0]))
		assert.Equal(t, []uuidStepKind{stepRepeatedMessage, stepStringLeaf}, stepKinds(paths[0]))
	})

	t.Run("mapMessageHopDescended", func(t *testing.T) {
		// A map<string, Msg> hop is descended into its value message.
		item := newMessage(t, "Item", stringField(t, "uuid", true))
		entry := mapEntryMessage(t, "ItemsByIdEntry", messageValueField(t, item))
		req := newMessage(t, "Req", mapField(t, "items_by_id", entry, false))
		paths := walkPathOf(t, req, entry, item)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"items_by_id", "uuid"}, stepNames(paths[0]))
		assert.Equal(t, []uuidStepKind{stepMapMessage, stepStringLeaf}, stepKinds(paths[0]))
	})

	t.Run("nestedContainersDescended", func(t *testing.T) {
		// A repeated message whose element carries a repeated string leaf
		// nests a container hop above a container leaf.
		item := newMessage(t, "Item", repeatedStringField(t, "actors", true))
		req := newMessage(t, "Req", repeatedMessageField(t, "items", item))
		paths := walkPathOf(t, req, item)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"items", "actors"}, stepNames(paths[0]))
		assert.Equal(t, []uuidStepKind{stepRepeatedMessage, stepRepeatedStringLeaf}, stepKinds(paths[0]))
	})

	t.Run("cycleThroughRepeatedHopSkipped", func(t *testing.T) {
		// A repeated self-reference must be pruned by the visited set just
		// like a scalar self-cycle; the annotated sibling wins.
		node := &protoplugin.Message{
			DescriptorProto: &descriptor.DescriptorProto{Name: proto.String("RepeatedCycleNode")},
		}
		node.Fields = []*protoplugin.Field{
			repeatedMessageField(t, "children", node),
			stringField(t, "id", true),
		}
		for _, f := range node.Fields {
			f.Message = node
		}
		paths := walkPathOf(t, node)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"id"}, stepNames(paths[0]))
	})

	t.Run("annotationOnMessageFieldIgnored", func(t *testing.T) {
		// Annotation directly on a message-typed field is ignored
		// (the leaf must be a string), but the walker still recurses
		// into the message looking for a string leaf inside.
		inner := newMessage(t, "Inner", stringField(t, "uuid", true))
		req := newMessage(t, "Req", annotatedMessageField(t, "creds", inner))
		paths := walkPathOf(t, req, inner)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"creds", "uuid"}, stepNames(paths[0]),
			"the annotation on creds is silently ignored, but the walker still descends into Inner and finds uuid")
	})

	t.Run("cycleWithSiblingAnnotation", func(t *testing.T) {
		// CycleNode references itself via `loop`. The walker must
		// mark CycleNode as visited before descending so it skips the
		// cyclic field and surfaces the annotated sibling `id`.
		cycle := &protoplugin.Message{
			DescriptorProto: &descriptor.DescriptorProto{Name: proto.String("CycleNode")},
		}
		cycle.Fields = []*protoplugin.Field{
			messageField(t, "loop", cycle),
			stringField(t, "id", true),
		}
		for _, f := range cycle.Fields {
			f.Message = cycle
		}
		paths := walkPathOf(t, cycle)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"id"}, stepNames(paths[0]),
			"the self-reference must be skipped via visited-set tracking; the annotated sibling wins")
	})

	t.Run("noAnnotationAnywhereReturnsNil", func(t *testing.T) {
		req := newMessage(t, "Req",
			stringField(t, "name", false),
			int64Field(t, "version", false),
		)
		assert.Empty(t, walkPathOf(t, req))
	})

	t.Run("emptyRequestReturnsNil", func(t *testing.T) {
		req := newMessage(t, "Req")
		assert.Empty(t, walkPathOf(t, req))
	})

	t.Run("messageFieldPointingNowhereDoesNotCrash", func(t *testing.T) {
		// A message-typed field whose TypeName resolves to no
		// registered message must be skipped gracefully (the walker
		// has no way to recurse, so it moves on to the next field).
		req := newMessage(t, "Req",
			&protoplugin.Field{FieldDescriptorProto: &descriptor.FieldDescriptorProto{
				Name:     proto.String("ghost"),
				Number:   proto.Int32(1),
				Type:     descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				Label:    descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				TypeName: proto.String(".svc.DoesNotExist"),
			}},
			stringField(t, "fallback", true),
		)
		paths := walkPathOf(t, req)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"fallback"}, stepNames(paths[0]))
	})
}

// TestBuildActorUUIDStmts asserts the structured builder statements lowered
// for a few representative paths. The statements carry no Go syntax - the
// template turns each into a line of code - so the assertions compare
// actorUUIDStmt values directly. The CamelCase normalisation is gogogen's,
// so snake_case field names map to PascalCase getters.
func TestBuildActorUUIDStmts(t *testing.T) {
	t.Run("directField", func(t *testing.T) {
		p := &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, "actor", true), Kind: stepStringLeaf},
		}}
		assert.Equal(t, []actorUUIDStmt{
			{Kind: stmtAppend, Expr: "t.GetActor()"},
		}, buildActorUUIDStmts(p))
	})

	t.Run("snakeCaseFieldGetsCamelCased", func(t *testing.T) {
		p := &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, "caller_actor_id", true), Kind: stepStringLeaf},
		}}
		assert.Equal(t, []actorUUIDStmt{
			{Kind: stmtAppend, Expr: "t.GetCallerActorId()"},
		}, buildActorUUIDStmts(p))
	})

	t.Run("multiStepScalarChain", func(t *testing.T) {
		p := &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, "outer", false), Kind: stepScalarMessage},
			{Field: stringField(t, "mid", false), Kind: stepScalarMessage},
			{Field: stringField(t, "inner_uuid", true), Kind: stepStringLeaf},
		}}
		assert.Equal(t, []actorUUIDStmt{
			{Kind: stmtAppend, Expr: "t.GetOuter().GetMid().GetInnerUuid()"},
		}, buildActorUUIDStmts(p))
	})

	t.Run("repeatedStringLeaf", func(t *testing.T) {
		p := &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, "actors", true), Kind: stepRepeatedStringLeaf},
		}}
		assert.Equal(t, []actorUUIDStmt{
			{Kind: stmtAppendSpread, Expr: "t.GetActors()"},
		}, buildActorUUIDStmts(p))
	})

	t.Run("mapStringLeaf", func(t *testing.T) {
		p := &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, "actors_by_role", true), Kind: stepMapStringLeaf},
		}}
		assert.Equal(t, []actorUUIDStmt{
			{Kind: stmtRangeOpen, Var: "v1", Expr: "t.GetActorsByRole()"},
			{Kind: stmtAppend, Expr: "v1"},
			{Kind: stmtClose},
		}, buildActorUUIDStmts(p))
	})

	t.Run("repeatedMessageHop", func(t *testing.T) {
		p := &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, "items", false), Kind: stepRepeatedMessage},
			{Field: stringField(t, "uuid", true), Kind: stepStringLeaf},
		}}
		assert.Equal(t, []actorUUIDStmt{
			{Kind: stmtRangeOpen, Var: "e1", Expr: "t.GetItems()"},
			{Kind: stmtAppend, Expr: "e1.GetUuid()"},
			{Kind: stmtClose},
		}, buildActorUUIDStmts(p))
	})

	t.Run("mapMessageHop", func(t *testing.T) {
		p := &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, "items_by_id", false), Kind: stepMapMessage},
			{Field: stringField(t, "uuid", true), Kind: stepStringLeaf},
		}}
		assert.Equal(t, []actorUUIDStmt{
			{Kind: stmtRangeOpen, Var: "e1", Expr: "t.GetItemsById()"},
			{Kind: stmtAppend, Expr: "e1.GetUuid()"},
			{Kind: stmtClose},
		}, buildActorUUIDStmts(p))
	})

	t.Run("nestedContainerLoops", func(t *testing.T) {
		// repeated Item where Item has a repeated string leaf: a container
		// hop above a container leaf nests one loop.
		p := &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, "items", false), Kind: stepRepeatedMessage},
			{Field: stringField(t, "actors", true), Kind: stepRepeatedStringLeaf},
		}}
		assert.Equal(t, []actorUUIDStmt{
			{Kind: stmtRangeOpen, Var: "e1", Expr: "t.GetItems()"},
			{Kind: stmtAppendSpread, Expr: "e1.GetActors()"},
			{Kind: stmtClose},
		}, buildActorUUIDStmts(p))
	})

	t.Run("repeatedMessageContainingMapStringLeaf", func(t *testing.T) {
		// repeated Item where Item has a map<string,string> leaf nests the
		// map's value loop inside the slice loop, with distinct variables.
		p := &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, "items", false), Kind: stepRepeatedMessage},
			{Field: stringField(t, "actors_by_role", true), Kind: stepMapStringLeaf},
		}}
		assert.Equal(t, []actorUUIDStmt{
			{Kind: stmtRangeOpen, Var: "e1", Expr: "t.GetItems()"},
			{Kind: stmtRangeOpen, Var: "v2", Expr: "e1.GetActorsByRole()"},
			{Kind: stmtAppend, Expr: "v2"},
			{Kind: stmtClose},
			{Kind: stmtClose},
		}, buildActorUUIDStmts(p))
	})
}

// TestNewActorUUIDMethod asserts the mode classification and the data each
// mode carries: scalar-only paths collapse to a single literal, a lone
// repeated-string leaf returns its slice directly, and any container hop
// forces the builder.
func TestNewActorUUIDMethod(t *testing.T) {
	scalar := func(name string) *uuidPath {
		return &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, name, true), Kind: stepStringLeaf},
		}}
	}

	t.Run("singleScalarIsLiteral", func(t *testing.T) {
		m := newActorUUIDMethod("Req", []*uuidPath{scalar("actor")})
		assert.Equal(t, modeLiteral, m.Mode)
		assert.Equal(t, []string{"t.GetActor()"}, m.Exprs)
	})

	t.Run("multipleScalarsShareOneLiteral", func(t *testing.T) {
		m := newActorUUIDMethod("Req", []*uuidPath{scalar("first"), scalar("second")})
		assert.Equal(t, modeLiteral, m.Mode)
		assert.Equal(t, []string{"t.GetFirst()", "t.GetSecond()"}, m.Exprs)
	})

	repeated := func(name string) *uuidPath {
		return &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, name, true), Kind: stepRepeatedStringLeaf},
		}}
	}

	t.Run("loneRepeatedStringLeafReturnsSliceDirectly", func(t *testing.T) {
		m := newActorUUIDMethod("Req", []*uuidPath{repeated("actors")})
		assert.Equal(t, modeSlice, m.Mode)
		assert.Equal(t, "t.GetActors()", m.SliceExpr,
			"a sole repeated-string leaf returns the getter's slice without a builder")
	})

	t.Run("loneRepeatedStringLeafThroughScalarHopReturnsChain", func(t *testing.T) {
		p := &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, "inner", false), Kind: stepScalarMessage},
			{Field: stringField(t, "actors", true), Kind: stepRepeatedStringLeaf},
		}}
		m := newActorUUIDMethod("Req", []*uuidPath{p})
		assert.Equal(t, modeSlice, m.Mode)
		assert.Equal(t, "t.GetInner().GetActors()", m.SliceExpr)
	})

	t.Run("multipleRepeatedStringLeavesFallBackToBuilder", func(t *testing.T) {
		// The direct-return shortcut only applies to a single path; two
		// slices must be concatenated, so the builder is used.
		m := newActorUUIDMethod("Req", []*uuidPath{repeated("actors"), repeated("admins")})
		assert.Equal(t, modeBuilder, m.Mode)
		assert.Equal(t, []actorUUIDStmt{
			{Kind: stmtAppendSpread, Expr: "t.GetActors()"},
			{Kind: stmtAppendSpread, Expr: "t.GetAdmins()"},
		}, m.Stmts)
	})

	t.Run("anyContainerForcesBuilderForAllPaths", func(t *testing.T) {
		// A scalar path mixed with a map leaf: the whole body must use
		// the builder style; the scalar still appends a single value.
		mapLeaf := &uuidPath{Steps: []uuidPathStep{
			{Field: stringField(t, "actors_by_role", true), Kind: stepMapStringLeaf},
		}}
		m := newActorUUIDMethod("Req", []*uuidPath{scalar("actor"), mapLeaf})
		assert.Equal(t, modeBuilder, m.Mode)
		assert.Equal(t, []actorUUIDStmt{
			{Kind: stmtAppend, Expr: "t.GetActor()"},
			{Kind: stmtRangeOpen, Var: "v1", Expr: "t.GetActorsByRole()"},
			{Kind: stmtAppend, Expr: "v1"},
			{Kind: stmtClose},
		}, m.Stmts)
	})
}

// TestActorUUIDMethods exercises the top-level template helper through
// synthetic services and messages, covering the dedup/skip rules.
func TestActorUUIDMethods(t *testing.T) {
	t.Run("emitsOnAnnotatedRequestType", func(t *testing.T) {
		req := newMessage(t, "DeleteUserRequest", stringField(t, "actor", true))
		info := newTemplateInfoWithServices(t,
			[]*protoplugin.Message{req},
			svc(t, "UserService", method(t, "DeleteUser", req, req)),
		)
		got, err := actorUUIDMethods(info)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "DeleteUserRequest", got[0].GoTypeName)
		assert.Equal(t, modeLiteral, got[0].Mode,
			"a scalar-only request collapses to a single return literal")
		assert.Equal(t, []string{"t.GetActor()"}, got[0].Exprs)
	})

	t.Run("collectsEveryAnnotatedFieldOnRequest", func(t *testing.T) {
		req := newMessage(t, "MultiRequest",
			stringField(t, "first_actor", true),
			stringField(t, "second_actor", true),
		)
		info := newTemplateInfoWithServices(t,
			[]*protoplugin.Message{req},
			svc(t, "S", method(t, "M", req, req)),
		)
		got, err := actorUUIDMethods(info)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "MultiRequest", got[0].GoTypeName)
		assert.Equal(t, modeLiteral, got[0].Mode)
		assert.Equal(t, []string{"t.GetFirstActor()", "t.GetSecondActor()"}, got[0].Exprs,
			"both annotated fields contribute an entry, in declaration order")
	})

	t.Run("dedupesWhenSameRequestUsedByMultipleMethods", func(t *testing.T) {
		req := newMessage(t, "Req", stringField(t, "actor", true))
		info := newTemplateInfoWithServices(t,
			[]*protoplugin.Message{req},
			svc(t, "S",
				method(t, "M1", req, req),
				method(t, "M2", req, req),
			),
		)
		got, err := actorUUIDMethods(info)
		require.NoError(t, err)
		assert.Len(t, got, 1, "Go does not allow two methods with the same name on the same receiver; the helper must dedupe")
	})

	t.Run("skipsRequestsWithoutAnnotatedPath", func(t *testing.T) {
		req := newMessage(t, "Req", stringField(t, "name", false))
		info := newTemplateInfoWithServices(t,
			[]*protoplugin.Message{req},
			svc(t, "S", method(t, "M", req, req)),
		)
		got, err := actorUUIDMethods(info)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("skipsAnnotatedMessageThatIsNotARequestType", func(t *testing.T) {
		// The accessor is emitted for method args only, mirroring
		// thriftrw-plugin-yarpc. A message that is directly annotated
		// (or reaches an annotation) but is never used as a service
		// method's request type must not get an accessor, even though it
		// lives in the file and the walker would happily find a path in
		// it. Regression guard against iterating all file messages
		// instead of just request types.
		req := newMessage(t, "DeleteUserRequest", stringField(t, "actor", true))
		// sidecar is annotated but only ever referenced as an
		// intermediate hop, never as a request type.
		sidecar := newMessage(t, "InnerLevel", stringField(t, "inner_uuid", true))
		info := newTemplateInfoWithServices(t,
			[]*protoplugin.Message{req, sidecar},
			svc(t, "UserService", method(t, "DeleteUser", req, req)),
		)
		got, err := actorUUIDMethods(info)
		require.NoError(t, err)
		require.Len(t, got, 1,
			"only the request type gets an accessor; the annotated non-request message must be skipped")
		assert.Equal(t, "DeleteUserRequest", got[0].GoTypeName)
	})

	t.Run("noopWhenExtensionNotInScope", func(t *testing.T) {
		// Target file has no actor_uuid extension reachable; the
		// helper short-circuits before walking anything.
		req := newMessage(t, "Req", stringField(t, "actor", true))
		target := &protoplugin.File{
			FileDescriptorProto: &descriptor.FileDescriptorProto{
				Name:    proto.String("svc/foo.proto"),
				Package: proto.String("svc"),
			},
			GoPackage: &protoplugin.GoPackage{Path: "svc/foopb"},
			Messages:  []*protoplugin.Message{req},
		}
		req.File = target
		got, err := actorUUIDMethods(&protoplugin.TemplateInfo{File: target})
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("rendersChainForNestedRequest", func(t *testing.T) {
		inner := newMessage(t, "Inner", stringField(t, "uuid", true))
		req := newMessage(t, "Outer", messageField(t, "inner", inner))
		info := newTemplateInfoWithServices(t,
			[]*protoplugin.Message{inner, req},
			svc(t, "S", method(t, "M", req, req)),
		)
		got, err := actorUUIDMethods(info)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "Outer", got[0].GoTypeName)
		assert.Equal(t, modeLiteral, got[0].Mode,
			"a scalar message chain collapses to a single return literal")
		assert.Equal(t, []string{"t.GetInner().GetUuid()"}, got[0].Exprs)
	})
}

// TestServiceAndMethodHasActorUUID exercises the server-template gating
// helpers: methodHasActorUUID reports per-method whether the request type
// reaches an annotation, and serviceHasActorUUID is the service-wide OR of
// that over every method. Both must stay in lockstep with actorUUIDMethods
// so the generated validator call (request.ActorUUID()) only appears where
// the accessor exists.
func TestServiceAndMethodHasActorUUID(t *testing.T) {
	annotatedReq := newMessage(t, "DeleteUserRequest", stringField(t, "actor", true))
	plainReq := newMessage(t, "PingRequest", stringField(t, "token", false))
	resp := newMessage(t, "Resp", stringField(t, "ok", false))

	deleteMethod := method(t, "DeleteUser", annotatedReq, resp)
	pingMethod := method(t, "Ping", plainReq, resp)

	info := newTemplateInfoWithServices(t,
		[]*protoplugin.Message{annotatedReq, plainReq, resp},
		svc(t, "UserService", deleteMethod, pingMethod),
	)

	t.Run("methodHasActorUUID_true_for_annotated_request", func(t *testing.T) {
		assert.True(t, methodHasActorUUID(info, deleteMethod))
	})

	t.Run("methodHasActorUUID_false_for_unannotated_request", func(t *testing.T) {
		assert.False(t, methodHasActorUUID(info, pingMethod))
	})

	t.Run("serviceHasActorUUID_true_when_any_method_annotated", func(t *testing.T) {
		assert.True(t, serviceHasActorUUID(info, info.File.Services[0]))
	})

	t.Run("serviceHasActorUUID_false_when_no_method_annotated", func(t *testing.T) {
		plainOnly := newTemplateInfoWithServices(t,
			[]*protoplugin.Message{plainReq, resp},
			svc(t, "PingService", method(t, "Ping", plainReq, resp)),
		)
		assert.False(t, serviceHasActorUUID(plainOnly, plainOnly.File.Services[0]))
	})

	t.Run("false_when_extension_not_in_scope", func(t *testing.T) {
		target := &protoplugin.File{
			FileDescriptorProto: &descriptor.FileDescriptorProto{
				Name:    proto.String("svc/foo.proto"),
				Package: proto.String("svc"),
			},
			GoPackage: &protoplugin.GoPackage{Path: "svc/foopb"},
		}
		annotatedReq.File = target
		s := svc(t, "S", method(t, "M", annotatedReq, resp))
		s.File = target
		target.Services = []*protoplugin.Service{s}
		noExt := &protoplugin.TemplateInfo{File: target}
		assert.False(t, serviceHasActorUUID(noExt, s),
			"without the extension in scope there is nothing to validate")
		assert.False(t, methodHasActorUUID(noExt, s.Methods[0]))
	})
}

// --- helpers --------------------------------------------------------------

// newOptionsFile builds a synthetic protoplugin.File that mirrors the shape
// of the monorepo's uber/auth/annotations/options.proto.
func newOptionsFile() *protoplugin.File {
	return &protoplugin.File{
		FileDescriptorProto: &descriptor.FileDescriptorProto{
			Name:    proto.String("uber/auth/annotations/options.proto"),
			Package: proto.String("uber.auth.annotations"),
			Extension: []*descriptor.FieldDescriptorProto{{
				Name:     proto.String("actor_uuid"),
				Number:   proto.Int32(_testActorUUIDFieldNumber),
				Type:     descriptor.FieldDescriptorProto_TYPE_BOOL.Enum(),
				Label:    descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Extendee: proto.String(fieldOptionsExtendee),
			}},
		},
	}
}

// withActorUUIDOption returns a *FieldOptions whose actor_uuid extension is
// set to v. Round-trips through proto.Marshal so the extension is stored as
// raw bytes - the same shape that hasActorUUID will see in a real plugin run.
func withActorUUIDOption(t *testing.T, v bool) *descriptor.FieldOptions {
	t.Helper()
	src := &descriptor.FieldOptions{}
	require.NoError(t, proto.SetExtension(src, _testActorUUIDExt, &v))
	bs, err := proto.Marshal(src)
	require.NoError(t, err)
	dst := &descriptor.FieldOptions{}
	require.NoError(t, proto.Unmarshal(bs, dst))
	return dst
}

// stringField builds a TYPE_STRING optional field. If annotated is true,
// the field carries the actor_uuid option set to true.
func stringField(t *testing.T, name string, annotated bool) *protoplugin.Field {
	t.Helper()
	fd := &descriptor.FieldDescriptorProto{
		Name:   proto.String(name),
		Number: proto.Int32(1),
		Type:   descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
		Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
	}
	if annotated {
		fd.Options = withActorUUIDOption(t, true)
	}
	return &protoplugin.Field{FieldDescriptorProto: fd}
}

// repeatedStringField builds a LABEL_REPEATED string field, optionally
// annotated. Used to confirm the walker skips list-shaped leaves.
func repeatedStringField(t *testing.T, name string, annotated bool) *protoplugin.Field {
	t.Helper()
	fd := &descriptor.FieldDescriptorProto{
		Name:   proto.String(name),
		Number: proto.Int32(1),
		Type:   descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
		Label:  descriptor.FieldDescriptorProto_LABEL_REPEATED.Enum(),
	}
	if annotated {
		fd.Options = withActorUUIDOption(t, true)
	}
	return &protoplugin.Field{FieldDescriptorProto: fd}
}

// repeatedInt64Field builds a LABEL_REPEATED int64 field, optionally
// annotated. Used to confirm the walker collects only string-valued
// containers and ignores repeated non-string scalars.
func repeatedInt64Field(t *testing.T, name string, annotated bool) *protoplugin.Field {
	t.Helper()
	fd := &descriptor.FieldDescriptorProto{
		Name:   proto.String(name),
		Number: proto.Int32(1),
		Type:   descriptor.FieldDescriptorProto_TYPE_INT64.Enum(),
		Label:  descriptor.FieldDescriptorProto_LABEL_REPEATED.Enum(),
	}
	if annotated {
		fd.Options = withActorUUIDOption(t, true)
	}
	return &protoplugin.Field{FieldDescriptorProto: fd}
}

// repeatedMessageField builds a LABEL_REPEATED message field whose type
// points at target. Used to exercise repeated-message hops.
func repeatedMessageField(t *testing.T, name string, target *protoplugin.Message) *protoplugin.Field {
	t.Helper()
	f := messageField(t, name, target)
	f.FieldDescriptorProto.Label = descriptor.FieldDescriptorProto_LABEL_REPEATED.Enum()
	return f
}

// stringValueField builds the `value` field (number 2) of a map entry
// whose value type is string.
func stringValueField(t *testing.T) *protoplugin.Field {
	t.Helper()
	return &protoplugin.Field{FieldDescriptorProto: &descriptor.FieldDescriptorProto{
		Name:   proto.String("value"),
		Number: proto.Int32(2),
		Type:   descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
		Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
	}}
}

// int64ValueField builds the `value` field of a map entry whose value
// type is int64 (a non-string value, used for the ignored-map case).
func int64ValueField(t *testing.T) *protoplugin.Field {
	t.Helper()
	return &protoplugin.Field{FieldDescriptorProto: &descriptor.FieldDescriptorProto{
		Name:   proto.String("value"),
		Number: proto.Int32(2),
		Type:   descriptor.FieldDescriptorProto_TYPE_INT64.Enum(),
		Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
	}}
}

// messageValueField builds the `value` field of a map entry whose value
// type is the message `target`.
func messageValueField(t *testing.T, target *protoplugin.Message) *protoplugin.Field {
	t.Helper()
	return &protoplugin.Field{FieldDescriptorProto: &descriptor.FieldDescriptorProto{
		Name:     proto.String("value"),
		Number:   proto.Int32(2),
		Type:     descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		Label:    descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		TypeName: proto.String(messageFQMN(target)),
	}}
}

// mapEntryMessage builds a synthetic proto map entry message: a message
// carrying the map_entry option with a string `key` (number 1) and the
// given `value` field (number 2). This mirrors what protoc synthesises
// for a map<K, V> field.
func mapEntryMessage(t *testing.T, name string, value *protoplugin.Field) *protoplugin.Message {
	t.Helper()
	key := &protoplugin.Field{FieldDescriptorProto: &descriptor.FieldDescriptorProto{
		Name:   proto.String("key"),
		Number: proto.Int32(1),
		Type:   descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
		Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
	}}
	m := newMessage(t, name, key, value)
	m.DescriptorProto.Options = &descriptor.MessageOptions{MapEntry: proto.Bool(true)}
	return m
}

// mapField builds a LABEL_REPEATED message field whose type is the map
// entry message `entry` (i.e. a proto map<K, V> field), optionally
// annotated with actor_uuid.
func mapField(t *testing.T, name string, entry *protoplugin.Message, annotated bool) *protoplugin.Field {
	t.Helper()
	fd := &descriptor.FieldDescriptorProto{
		Name:     proto.String(name),
		Number:   proto.Int32(1),
		Type:     descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
		Label:    descriptor.FieldDescriptorProto_LABEL_REPEATED.Enum(),
		TypeName: proto.String(messageFQMN(entry)),
	}
	if annotated {
		fd.Options = withActorUUIDOption(t, true)
	}
	return &protoplugin.Field{FieldDescriptorProto: fd}
}

// int64Field builds an int64 optional field, optionally annotated. Used
// to confirm the walker rejects non-string leaves.
func int64Field(t *testing.T, name string, annotated bool) *protoplugin.Field {
	t.Helper()
	fd := &descriptor.FieldDescriptorProto{
		Name:   proto.String(name),
		Number: proto.Int32(1),
		Type:   descriptor.FieldDescriptorProto_TYPE_INT64.Enum(),
		Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
	}
	if annotated {
		fd.Options = withActorUUIDOption(t, true)
	}
	return &protoplugin.Field{FieldDescriptorProto: fd}
}

// messageField builds a TYPE_MESSAGE optional field whose type points at
// `target`. Used to chain messages in walker tests.
func messageField(t *testing.T, name string, target *protoplugin.Message) *protoplugin.Field {
	t.Helper()
	return &protoplugin.Field{
		FieldDescriptorProto: &descriptor.FieldDescriptorProto{
			Name:     proto.String(name),
			Number:   proto.Int32(1),
			Type:     descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
			Label:    descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			TypeName: proto.String(messageFQMN(target)),
		},
	}
}

// annotatedMessageField is messageField with the actor_uuid option set
// to true on the field itself. Used to confirm the walker silently
// ignores the misplaced annotation but still recurses into target.
func annotatedMessageField(t *testing.T, name string, target *protoplugin.Message) *protoplugin.Field {
	t.Helper()
	f := messageField(t, name, target)
	f.FieldDescriptorProto.Options = withActorUUIDOption(t, true)
	return f
}

// messageFQMN returns the FQMN that walkForUUID's lookup expects:
// ".pkg.MessageName" for a top-level message in the synthetic test
// package "svc".
func messageFQMN(m *protoplugin.Message) string {
	if m == nil {
		return ""
	}
	// In the test scaffolding, every synthetic message is top-level in
	// package "svc"; m.FQMN() would require m.File which not all tests
	// initialise, so we hand-roll the same shape here.
	return ".svc." + m.GetName()
}

// newMessage builds a top-level message with the given fields wired
// to back-reference the message.
func newMessage(t *testing.T, name string, fields ...*protoplugin.Field) *protoplugin.Message {
	t.Helper()
	m := &protoplugin.Message{
		DescriptorProto: &descriptor.DescriptorProto{Name: proto.String(name)},
		Fields:          fields,
	}
	for _, f := range fields {
		f.Message = m
	}
	return m
}

// walkPathOf wraps walkForUUID with a fresh context indexing the given
// messages and a fresh visited set seeded with `req`. Mirrors the entry
// path actorUUIDMethods uses, but bypasses the file/service scaffolding.
func walkPathOf(t *testing.T, req *protoplugin.Message, others ...*protoplugin.Message) []*uuidPath {
	t.Helper()
	all := append([]*protoplugin.Message{req}, others...)
	file := &protoplugin.File{
		FileDescriptorProto: &descriptor.FileDescriptorProto{
			Name:    proto.String("svc/foo.proto"),
			Package: proto.String("svc"),
		},
		Messages: all,
	}
	for _, m := range all {
		m.File = file
	}
	ctx := newUUIDContext(file)
	return walkForUUID(req.Fields, _testActorUUIDFieldNumber, ctx, map[*protoplugin.Message]bool{req: true})
}

// svc builds a synthetic *protoplugin.Service with the given methods.
// Methods' parent service back-pointer is wired so the helper's
// iteration mirrors a real registry-loaded service.
func svc(t *testing.T, name string, methods ...*protoplugin.Method) *protoplugin.Service {
	t.Helper()
	s := &protoplugin.Service{
		ServiceDescriptorProto: &descriptor.ServiceDescriptorProto{Name: proto.String(name)},
		Methods:                methods,
	}
	for _, m := range methods {
		m.Service = s
	}
	return s
}

// method builds a synthetic *protoplugin.Method with the given request
// and response message types resolved.
func method(t *testing.T, name string, req, resp *protoplugin.Message) *protoplugin.Method {
	t.Helper()
	return &protoplugin.Method{
		MethodDescriptorProto: &descriptor.MethodDescriptorProto{
			Name:       proto.String(name),
			InputType:  proto.String(messageFQMN(req)),
			OutputType: proto.String(messageFQMN(resp)),
		},
		RequestType:  req,
		ResponseType: resp,
	}
}

// newTemplateInfoWithServices wires messages and services into a
// synthetic file that transitively imports the options.proto, so the
// extension is resolvable and actorUUIDMethods walks normally.
func newTemplateInfoWithServices(t *testing.T, messages []*protoplugin.Message, services ...*protoplugin.Service) *protoplugin.TemplateInfo {
	t.Helper()
	target := &protoplugin.File{
		FileDescriptorProto: &descriptor.FileDescriptorProto{
			Name:    proto.String("svc/foo.proto"),
			Package: proto.String("svc"),
		},
		GoPackage:              &protoplugin.GoPackage{Path: "svc/foopb"},
		Messages:               messages,
		Services:               services,
		TransitiveDependencies: []*protoplugin.File{newOptionsFile()},
	}
	for _, m := range messages {
		m.File = target
	}
	for _, s := range services {
		s.File = target
	}
	return &protoplugin.TemplateInfo{File: target}
}
