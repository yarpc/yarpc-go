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
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/yarpc/internal/protogen"
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
	Filename:      "uber/security/engsec/utoken/annotations/options_test.proto",
}

// _testUnrelatedExtFieldNumber hosts an extension that is NOT actor_uuid,
// simulating the collision scenario: the descriptor set assigns actor_uuid
// a field number at which the plugin binary's registry already holds an
// unrelated extension (as gogoproto's init() does for 65001-65013).
const _testUnrelatedExtFieldNumber = 99998

var _testUnrelatedExt = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FieldOptions)(nil),
	ExtensionType: (*string)(nil),
	Field:         _testUnrelatedExtFieldNumber,
	Name:          "some.other.pkg.custom_option",
	Tag:           "bytes,99998,opt,name=custom_option",
	Filename:      "some/other/pkg/options_test.proto",
}

// _testWrongTypeExtFieldNumber hosts an extension registered under
// actor_uuid's own name but with a non-bool type, exercising the type leg
// of the registry validation.
const _testWrongTypeExtFieldNumber = 99997

var _testWrongTypeExt = &proto.ExtensionDesc{
	ExtendedType:  (*descriptor.FieldOptions)(nil),
	ExtensionType: (*string)(nil),
	Field:         _testWrongTypeExtFieldNumber,
	Name:          _ActorUUIDFQN,
	Tag:           "bytes,99997,opt,name=actor_uuid",
	Filename:      "uber/security/engsec/utoken/annotations/options_test.proto",
}

func init() {
	proto.RegisterExtension(_testActorUUIDExt)
	proto.RegisterExtension(_testUnrelatedExt)
	proto.RegisterExtension(_testWrongTypeExt)
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
				Name:    proto.String("uber/security/engsec/utoken/annotations/options.proto"),
				Package: proto.String("uber.security.engsec.utoken.annotations"),
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
		got, err := hasActorUUID(opts, _testActorUUIDFieldNumber)
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("explicit_false", func(t *testing.T) {
		opts := withActorUUIDOption(t, false)
		got, err := hasActorUUID(opts, _testActorUUIDFieldNumber)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("not_set", func(t *testing.T) {
		got, err := hasActorUUID(&descriptor.FieldOptions{}, _testActorUUIDFieldNumber)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("nil_options", func(t *testing.T) {
		got, err := hasActorUUID(nil, _testActorUUIDFieldNumber)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("zero_field_number", func(t *testing.T) {
		opts := withActorUUIDOption(t, true)
		got, err := hasActorUUID(opts, 0)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("unrelated_extension_at_number_errors", func(t *testing.T) {
		// The registry entry at this number is a different extension
		// entirely: the lookup must fail loudly, never silently report
		// the field as unannotated.
		got, err := hasActorUUID(withActorUUIDOption(t, true), _testUnrelatedExtFieldNumber)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "collision")
		assert.Contains(t, err.Error(), "some.other.pkg.custom_option")
		assert.False(t, got)
	})

	t.Run("wrong_typed_extension_at_number_errors", func(t *testing.T) {
		// Right name, wrong Go type: reading it as a bool would be a
		// wiretype mismatch, so the descriptor is rejected up front.
		got, err := hasActorUUID(withActorUUIDOption(t, true), _testWrongTypeExtFieldNumber)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected *bool")
		assert.False(t, got)
	})
}

// stepNames returns the root-to-leaf proto field names of the path.
// Mirrors the helper in encoding/thrift/thriftrw-plugin-yarpc/uuid_test.go.
func stepNames(p *protogen.Path) []string {
	if p == nil {
		return nil
	}
	out := make([]string, 0, len(p.Steps))
	for _, s := range p.Steps {
		out = append(out, s.Name)
	}
	return out
}

// allStepNames flattens the per-path step names of a walk result so a
// multi-path expectation can be asserted in one comparison.
func allStepNames(paths []*protogen.Path) [][]string {
	out := make([][]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, stepNames(p))
	}
	return out
}

// stepKinds returns the per-step traversal kinds of the path, used to
// assert that container hops/leaves are classified correctly.
func stepKinds(p *protogen.Path) []protogen.StepKind {
	if p == nil {
		return nil
	}
	out := make([]protogen.StepKind, 0, len(p.Steps))
	for _, s := range p.Steps {
		out = append(out, s.Kind)
	}
	return out
}

// TestConvertAndWalk exercises the converter and the shared protogen
// walker together, using synthetic in-memory descriptors so each subtest
// is independent of the runtime gogo proto registry. The classification
// (shapes, annotation reads, map entries) lives in the converter; the
// traversal semantics (order, cycles, capping) live in protogen.Walk,
// which has its own literal-graph tests. Mirrors the structure of
// encoding/thrift/thriftrw-plugin-yarpc/uuid_test.go's TestUUIDPathInArgs.
func TestConvertAndWalk(t *testing.T) {
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
		assert.Equal(t, []protogen.StepKind{protogen.StepRepeatedStringLeaf}, stepKinds(paths[0]))
	})

	t.Run("mapStringLeafCollected", func(t *testing.T) {
		// A map<string, string> carrying the annotation is a valid leaf:
		// every value is collected.
		entry := mapEntryMessage(t, "ActorsByRoleEntry", stringValueField(t))
		req := newMessage(t, "Req", mapField(t, "actors_by_role", entry, true))
		paths := walkPathOf(t, req, entry)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"actors_by_role"}, stepNames(paths[0]))
		assert.Equal(t, []protogen.StepKind{protogen.StepMapStringLeaf}, stepKinds(paths[0]))
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
		assert.Equal(t, []protogen.StepKind{protogen.StepRepeatedMessage, protogen.StepStringLeaf}, stepKinds(paths[0]))
	})

	t.Run("mapMessageHopDescended", func(t *testing.T) {
		// A map<string, Msg> hop is descended into its value message.
		item := newMessage(t, "Item", stringField(t, "uuid", true))
		entry := mapEntryMessage(t, "ItemsByIdEntry", messageValueField(t, item))
		req := newMessage(t, "Req", mapField(t, "items_by_id", entry, false))
		paths := walkPathOf(t, req, entry, item)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"items_by_id", "uuid"}, stepNames(paths[0]))
		assert.Equal(t, []protogen.StepKind{protogen.StepMapMessage, protogen.StepStringLeaf}, stepKinds(paths[0]))
	})

	t.Run("nestedContainersDescended", func(t *testing.T) {
		// A repeated message whose element carries a repeated string leaf
		// nests a container hop above a container leaf.
		item := newMessage(t, "Item", repeatedStringField(t, "actors", true))
		req := newMessage(t, "Req", repeatedMessageField(t, "items", item))
		paths := walkPathOf(t, req, item)
		require.Len(t, paths, 1)
		assert.Equal(t, []string{"items", "actors"}, stepNames(paths[0]))
		assert.Equal(t, []protogen.StepKind{protogen.StepRepeatedMessage, protogen.StepRepeatedStringLeaf}, stepKinds(paths[0]))
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

// The mode classification and statement lowering tests live with the
// shared implementation in internal/protogen (TestLower and
// TestBuildStmts): lowering consumes only neutral Step values, so its
// tests need no descriptors. newActorUUIDMethod itself is covered
// through TestActorUUIDMethods below, which asserts the wired-up
// GoTypeName/Mode/Exprs on real converter+walk output.

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
		assert.Equal(t, protogen.ModeLiteral, got[0].Mode,
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
		assert.Equal(t, protogen.ModeLiteral, got[0].Mode)
		assert.Equal(t, []string{"t.GetFirstActor()", "t.GetSecondActor()"}, got[0].Exprs,
			"both annotated fields contribute an entry, in declaration order")
	})

	t.Run("emitsOncePerMessageRegardlessOfMethodCount", func(t *testing.T) {
		// Emission iterates declared messages, so a request type shared
		// by many methods contributes exactly one accessor (Go does not
		// allow two methods with the same name on the same receiver).
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
		assert.Len(t, got, 1)
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

	t.Run("emitsOnAnnotatedNonRequestMessage", func(t *testing.T) {
		// Emission is keyed on declaration, not service usage: a message
		// with an annotated path gets an accessor even when no method
		// uses it as a request type. The declaring file cannot know who
		// uses its messages (the service may live in another file or
		// package), so it must emit for every annotated message.
		req := newMessage(t, "DeleteUserRequest", stringField(t, "actor", true))
		sidecar := newMessage(t, "InnerLevel", stringField(t, "inner_uuid", true))
		info := newTemplateInfoWithServices(t,
			[]*protoplugin.Message{req, sidecar},
			svc(t, "UserService", method(t, "DeleteUser", req, req)),
		)
		got, err := actorUUIDMethods(info)
		require.NoError(t, err)
		require.Len(t, got, 2,
			"every declared message with an annotated path gets an accessor")
		assert.Equal(t, "DeleteUserRequest", got[0].GoTypeName)
		assert.Equal(t, "InnerLevel", got[1].GoTypeName)
		assert.Equal(t, []string{"t.GetInnerUuid()"}, got[1].Exprs)
	})

	t.Run("emitsForFileWithoutServices", func(t *testing.T) {
		// The types.proto layout: a file declaring only messages, whose
		// services live elsewhere. Its generation must still emit the
		// accessor - this is where the method can legally be defined.
		req := newMessage(t, "DeleteUserRequest", stringField(t, "actor", true))
		info := newTemplateInfoWithServices(t, []*protoplugin.Message{req})
		got, err := actorUUIDMethods(info)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "DeleteUserRequest", got[0].GoTypeName)
	})

	t.Run("emitsInDeclaringFileNotInServiceFile", func(t *testing.T) {
		// Split layout: the request type is declared in types.proto, the
		// service in service.proto. The accessor must come from the
		// declaring file's generation; the service file declares no
		// messages and must emit nothing.
		req := newMessage(t, "DeleteUserRequest", stringField(t, "actor", true))
		typesFile := &protoplugin.File{
			FileDescriptorProto: &descriptor.FileDescriptorProto{
				Name:    proto.String("svc/types.proto"),
				Package: proto.String("svc"),
			},
			GoPackage:              &protoplugin.GoPackage{Path: "svc/foopb"},
			Messages:               []*protoplugin.Message{req},
			TransitiveDependencies: []*protoplugin.File{newOptionsFile()},
		}
		req.File = typesFile
		serviceFile := &protoplugin.File{
			FileDescriptorProto: &descriptor.FileDescriptorProto{
				Name:    proto.String("svc/service.proto"),
				Package: proto.String("svc"),
			},
			GoPackage:              &protoplugin.GoPackage{Path: "svc/foopb"},
			Services:               []*protoplugin.Service{svc(t, "UserService", method(t, "DeleteUser", req, req))},
			TransitiveDependencies: []*protoplugin.File{typesFile, newOptionsFile()},
		}

		fromTypes, err := actorUUIDMethods(&protoplugin.TemplateInfo{File: typesFile})
		require.NoError(t, err)
		require.Len(t, fromTypes, 1,
			"the declaring file emits the accessor even with no services in it")
		assert.Equal(t, "DeleteUserRequest", fromTypes[0].GoTypeName)

		fromService, err := actorUUIDMethods(&protoplugin.TemplateInfo{File: serviceFile})
		require.NoError(t, err)
		assert.Empty(t, fromService,
			"the service file declares no messages and must not re-emit the accessor")
	})

	t.Run("skipsSyntheticMapEntryMessages", func(t *testing.T) {
		// Map entry messages appear in the file's message list but gogo
		// generates no Go struct for them; they must never get a method,
		// even when their value field would walk to a leaf.
		item := newMessage(t, "Item", stringField(t, "uuid", true))
		entry := mapEntryMessage(t, "ItemsByIdEntry", messageValueField(t, item))
		req := newMessage(t, "Req", mapField(t, "items_by_id", entry, false))
		info := newTemplateInfoWithServices(t,
			[]*protoplugin.Message{req, entry, item},
			svc(t, "S", method(t, "M", req, req)),
		)
		got, err := actorUUIDMethods(info)
		require.NoError(t, err)
		require.Len(t, got, 2, "Req and Item get accessors; the map entry must not")
		assert.Equal(t, "Req", got[0].GoTypeName)
		assert.Equal(t, "Item", got[1].GoTypeName)
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

	t.Run("pathCountUnderCapSucceeds", func(t *testing.T) {
		// 6 chained diamonds expand to 2^6 = 64 paths, under the cap of
		// 100: generation succeeds and every route gets its expression.
		// Every message on the chain reaches the leaf, so each declared
		// message gets its own accessor (root, the leaf itself, and the
		// six diamond levels).
		root, rest := diamondChain(t, 6)
		info := newTemplateInfoWithServices(t,
			append([]*protoplugin.Message{root}, rest...),
			svc(t, "S", method(t, "M", root, root)),
		)
		got, err := actorUUIDMethods(info)
		require.NoError(t, err)
		require.Len(t, got, 8)
		assert.Equal(t, "Req", got[0].GoTypeName)
		assert.Len(t, got[0].Exprs, 64, "each of the 2^6 routes contributes one expression")
	})

	t.Run("pathCountOverCapFailsGeneration", func(t *testing.T) {
		// 7 chained diamonds expand to 2^7 = 128 paths, over the cap of
		// 100: generation must fail with a clear error instead of
		// emitting an enormous accessor.
		root, rest := diamondChain(t, 7)
		info := newTemplateInfoWithServices(t,
			append([]*protoplugin.Message{root}, rest...),
			svc(t, "S", method(t, "M", root, root)),
		)
		_, err := actorUUIDMethods(info)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "actor_uuid paths")
		assert.Contains(t, err.Error(), "Req")
	})

	t.Run("registryCollisionFailsGeneration", func(t *testing.T) {
		// The descriptor set declares actor_uuid at a field number the
		// plugin binary's extension registry already assigns to an
		// unrelated extension. Generation must fail loudly: silently
		// emitting no accessors would drop the security annotation.
		req := newMessage(t, "DeleteUserRequest", stringField(t, "actor", true))
		target := &protoplugin.File{
			FileDescriptorProto: &descriptor.FileDescriptorProto{
				Name:    proto.String("svc/foo.proto"),
				Package: proto.String("svc"),
			},
			GoPackage:              &protoplugin.GoPackage{Path: "svc/foopb"},
			Messages:               []*protoplugin.Message{req},
			Services:               []*protoplugin.Service{svc(t, "UserService", method(t, "DeleteUser", req, req))},
			TransitiveDependencies: []*protoplugin.File{newOptionsFileAt(_testUnrelatedExtFieldNumber)},
		}
		req.File = target
		_, err := actorUUIDMethods(&protoplugin.TemplateInfo{File: target})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "collision")
		assert.Contains(t, err.Error(), "some.other.pkg.custom_option")
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
		require.Len(t, got, 2, "both Inner (declares the leaf) and Outer (reaches it) get accessors")
		assert.Equal(t, "Inner", got[0].GoTypeName)
		assert.Equal(t, []string{"t.GetUuid()"}, got[0].Exprs)
		assert.Equal(t, "Outer", got[1].GoTypeName)
		assert.Equal(t, protogen.ModeLiteral, got[1].Mode,
			"a scalar message chain collapses to a single return literal")
		assert.Equal(t, []string{"t.GetInner().GetUuid()"}, got[1].Exprs)
	})
}

// --- helpers --------------------------------------------------------------

// newOptionsFile builds a synthetic protoplugin.File that mirrors the shape
// of the monorepo's uber/security/engsec/utoken/annotations/options.proto.
func newOptionsFile() *protoplugin.File {
	return newOptionsFileAt(_testActorUUIDFieldNumber)
}

// newOptionsFileAt is newOptionsFile with the actor_uuid extension declared
// at an arbitrary field number, used to simulate a declaration colliding
// with an unrelated extension in the plugin binary's registry.
func newOptionsFileAt(num int32) *protoplugin.File {
	return &protoplugin.File{
		FileDescriptorProto: &descriptor.FileDescriptorProto{
			Name:    proto.String("uber/security/engsec/utoken/annotations/options.proto"),
			Package: proto.String("uber.security.engsec.utoken.annotations"),
			Extension: []*descriptor.FieldDescriptorProto{{
				Name:     proto.String("actor_uuid"),
				Number:   proto.Int32(num),
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

// walkPathOf converts `req` to the neutral protogen graph (with a fresh
// context indexing the given messages) and walks it. Mirrors the entry
// path actorUUIDMethods uses, but bypasses the file/service scaffolding.
func walkPathOf(t *testing.T, req *protoplugin.Message, others ...*protoplugin.Message) []*protogen.Path {
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
	node, err := newUUIDConverter(_testActorUUIDFieldNumber, ctx).convert(req)
	require.NoError(t, err)
	paths, _ := protogen.Walk(node, _maxActorUUIDPaths)
	return paths
}

// diamondChain builds a chain of `levels` diamond-shaped messages: each
// level has two fields (`a`, `b`) of the next level's type, and the last
// level funnels into a single leaf message with an annotated uuid field.
// A request walking the chain expands to 2^levels paths. Returns the root
// message followed by every other message needed to index the context.
func diamondChain(t *testing.T, levels int) (*protoplugin.Message, []*protoplugin.Message) {
	t.Helper()
	leaf := newMessage(t, "Leaf", stringField(t, "uuid", true))
	all := []*protoplugin.Message{leaf}
	next := leaf
	for i := levels; i >= 1; i-- {
		m := newMessage(t, fmt.Sprintf("D%d", i),
			messageField(t, "a", next),
			messageField(t, "b", next),
		)
		all = append(all, m)
		next = m
	}
	root := newMessage(t, "Req", messageField(t, "root", next))
	return root, all
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
