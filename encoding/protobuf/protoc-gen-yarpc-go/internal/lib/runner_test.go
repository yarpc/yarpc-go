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
	plugin_go "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunnerEmitsActorUUIDAccessor drives the full plugin pipeline against an
// in-memory CodeGeneratorRequest and asserts the generated .pb.yarpc.go
// contains an ActorUUID() accessor for the annotated request message.
//
// This is the integration-level coverage for the (uber.auth.annotations.
// actor_uuid) annotation: it exercises template compilation, helper wiring,
// extension discovery from descriptor, and gofmt of the generated source.
func TestRunnerEmitsActorUUIDAccessor(t *testing.T) {
	req := &plugin_go.CodeGeneratorRequest{
		FileToGenerate: []string{"svc/foo.proto"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			optionsFileDescriptor(),
			targetFileDescriptor(t),
		},
	}

	resp := Runner.Run(req)
	require.Nil(t, resp.Error, "plugin returned error: %v", resp.Error)
	require.Len(t, resp.File, 1)

	out := resp.File[0].GetContent()

	assert.Contains(t, out, "func (t *DeleteUserRequest) ActorUUID() []string {",
		"missing accessor on annotated message; output was:\n%s", out)
	assert.Contains(t, out, "t.GetActor()")

	assert.NotContains(t, out, "func (t *DeleteUserResponse) ActorUUID()",
		"unannotated message must not get an accessor")

	assert.Contains(t, out, "type UserServiceYARPCClient interface",
		"sanity check: existing client/server emission still works")
}

// TestRunnerWiresActorUUIDValidator asserts the server template injects the
// ActorUUID validator plumbing for a service with an annotated request
// type: the handler carries a validator field, Build/registration accept
// protobuf.RegisterOptions, and the per-method handler calls
// protobuf.ValidateActorUUID with the request's ActorUUID() accessor.
func TestRunnerWiresActorUUIDValidator(t *testing.T) {
	req := &plugin_go.CodeGeneratorRequest{
		FileToGenerate: []string{"svc/foo.proto"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			optionsFileDescriptor(),
			targetFileDescriptor(t),
		},
	}

	resp := Runner.Run(req)
	require.Nil(t, resp.Error, "plugin returned error: %v", resp.GetError())
	require.Len(t, resp.File, 1)

	out := resp.File[0].GetContent()
	assert.Contains(t, out, "actorUUIDValidator protobuf.ActorUUIDValidator",
		"handler struct must carry a validator field; output was:\n%s", out)
	assert.Contains(t, out, "func BuildUserServiceYARPCProcedures(server UserServiceYARPCServer, options ...protobuf.RegisterOption)",
		"Build entry point must accept register options; output was:\n%s", out)
	assert.Contains(t, out, "protobuf.ActorUUIDValidatorFromOptions(options)",
		"Build entry point must extract the validator from options; output was:\n%s", out)
	assert.Contains(t, out, `protobuf.ValidateActorUUID(ctx, h.actorUUIDValidator, request.ActorUUID(), "svc.UserService", "DeleteUser")`,
		"the annotated method must call the validator with its ActorUUID(); output was:\n%s", out)
}

// TestRunnerSkipsValidatorForUnannotatedService asserts that a service
// whose request types carry no annotation keeps its original, option-free
// signature and gets no validator plumbing, preserving backwards
// compatibility.
func TestRunnerSkipsValidatorForUnannotatedService(t *testing.T) {
	target := targetFileDescriptor(t)
	// Drop the annotation from the request's first field.
	target.MessageType[0].Field[0].Options = nil

	req := &plugin_go.CodeGeneratorRequest{
		FileToGenerate: []string{"svc/foo.proto"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			optionsFileDescriptor(),
			target,
		},
	}

	resp := Runner.Run(req)
	require.Nil(t, resp.Error, "plugin returned error: %v", resp.GetError())
	require.Len(t, resp.File, 1)

	out := resp.File[0].GetContent()
	assert.NotContains(t, out, "actorUUIDValidator",
		"an unannotated service must not get validator plumbing; output was:\n%s", out)
	assert.NotContains(t, out, "ValidateActorUUID",
		"an unannotated service must not call the validator; output was:\n%s", out)
	assert.Contains(t, out, "func BuildUserServiceYARPCProcedures(server UserServiceYARPCServer) []transport.Procedure",
		"an unannotated service keeps its original option-free signature; output was:\n%s", out)
}

// TestRunnerCollectsEveryAnnotationOnMultiplyAnnotatedRequest asserts the
// "collect all" rule: a request type with two actor_uuid-annotated fields
// generates exactly one ActorUUID() accessor whose []string body returns
// both annotated fields, in declaration order. Reaching more than one
// annotation is not an error.
func TestRunnerCollectsEveryAnnotationOnMultiplyAnnotatedRequest(t *testing.T) {
	target := targetFileDescriptor(t)
	// Add a second annotated field after the existing first one.
	deleteReq := target.MessageType[0]
	deleteReq.Field = append(deleteReq.Field, &descriptor.FieldDescriptorProto{
		Name:    proto.String("actor_b"),
		Number:  proto.Int32(3),
		Type:    descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
		Label:   descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Options: withActorUUIDOption(t, true),
	})

	req := &plugin_go.CodeGeneratorRequest{
		FileToGenerate: []string{"svc/foo.proto"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			optionsFileDescriptor(),
			target,
		},
	}

	resp := Runner.Run(req)
	require.Nil(t, resp.Error, "plugin returned error: %v", resp.GetError())
	require.Len(t, resp.File, 1)

	out := resp.File[0].GetContent()
	assert.Contains(t, out, "func (t *DeleteUserRequest) ActorUUID() []string {",
		"missing accessor on annotated message; output was:\n%s", out)
	assert.Contains(t, out, "t.GetActor()",
		"the first annotated field must contribute an entry; output was:\n%s", out)
	assert.Contains(t, out, "t.GetActorB()",
		"the second annotation must also be collected; output was:\n%s", out)
}

// TestRunnerEmitsActorUUIDAccessorForNestedRequest drives the path
// walker through the full plugin pipeline: the request type has no
// directly-annotated field but reaches one through a message hop. The
// generated accessor must chain GetXxx() calls down to the leaf.
func TestRunnerEmitsActorUUIDAccessorForNestedRequest(t *testing.T) {
	target := nestedTargetFileDescriptor(t)
	req := &plugin_go.CodeGeneratorRequest{
		FileToGenerate: []string{"svc/foo.proto"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			optionsFileDescriptor(),
			target,
		},
	}

	resp := Runner.Run(req)
	require.Nil(t, resp.Error, "plugin returned error: %v", resp.GetError())
	require.Len(t, resp.File, 1)

	out := resp.File[0].GetContent()
	assert.Contains(t, out, "func (t *Outer) ActorUUID() []string {",
		"missing accessor on the request type; output was:\n%s", out)
	assert.Contains(t, out, "t.GetInner().GetUuid()",
		"chain must walk from request through the nested message hop; output was:\n%s", out)
	assert.NotContains(t, out, "func (t *Inner) ActorUUID()",
		"only the request type gets the accessor; nested types must not be re-emitted; output was:\n%s", out)
}

// TestRunnerEmitsActorUUIDAccessorForContainers drives the walker and
// codegen through every container shape end-to-end: a repeated string
// leaf, a map<string,string> leaf, a repeated-message hop, and a
// map<string,Msg> hop. The generated accessor must use the builder style
// with the appropriate append / range statements for each.
func TestRunnerEmitsActorUUIDAccessorForContainers(t *testing.T) {
	target := containerTargetFileDescriptor(t)
	req := &plugin_go.CodeGeneratorRequest{
		FileToGenerate: []string{"svc/foo.proto"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			optionsFileDescriptor(),
			target,
		},
	}

	resp := Runner.Run(req)
	require.Nil(t, resp.Error, "plugin returned error: %v", resp.GetError())
	require.Len(t, resp.File, 1)

	out := resp.File[0].GetContent()
	assert.Contains(t, out, "func (t *Container) ActorUUID() []string {",
		"missing accessor on the container request type; output was:\n%s", out)
	assert.Contains(t, out, "out = append(out, t.GetActors()...)",
		"repeated string leaf must append all elements; output was:\n%s", out)
	assert.Contains(t, out, "for _, v1 := range t.GetActorsByRole() {",
		"map<string,string> leaf must range over values; output was:\n%s", out)
	assert.Contains(t, out, "for _, e1 := range t.GetItems() {",
		"repeated message hop must range over the slice; output was:\n%s", out)
	assert.Contains(t, out, "for _, e1 := range t.GetItemsById() {",
		"map<string,Msg> hop must range over the values; output was:\n%s", out)
	assert.Contains(t, out, "out = append(out, e1.GetUuid())",
		"message-hop elements must recurse to the annotated leaf; output was:\n%s", out)
}

// containerTargetFileDescriptor builds a synthetic svc/foo.proto whose
// request type Container exercises every container shape the walker now
// supports: a repeated string leaf, a map<string,string> leaf, a
// repeated-message hop, and a map<string,Item> hop, where Item carries an
// annotated string leaf. Map fields are encoded as the synthetic
// map_entry nested messages protoc would generate.
func containerTargetFileDescriptor(t *testing.T) *descriptor.FileDescriptorProto {
	t.Helper()
	mapEntry := func(name string, value *descriptor.FieldDescriptorProto) *descriptor.DescriptorProto {
		return &descriptor.DescriptorProto{
			Name:    proto.String(name),
			Options: &descriptor.MessageOptions{MapEntry: proto.Bool(true)},
			Field: []*descriptor.FieldDescriptorProto{
				{
					Name:   proto.String("key"),
					Number: proto.Int32(1),
					Type:   descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
					Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				},
				value,
			},
		}
	}
	return &descriptor.FileDescriptorProto{
		Name:       proto.String("svc/foo.proto"),
		Package:    proto.String("svc"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"uber/auth/annotations/options.proto"},
		Options: &descriptor.FileOptions{
			GoPackage: proto.String("svc/foopb"),
		},
		MessageType: []*descriptor.DescriptorProto{
			{
				Name: proto.String("Item"),
				Field: []*descriptor.FieldDescriptorProto{{
					Name:    proto.String("uuid"),
					Number:  proto.Int32(1),
					Type:    descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
					Label:   descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Options: withActorUUIDOption(t, true),
				}},
			},
			{
				Name: proto.String("Container"),
				NestedType: []*descriptor.DescriptorProto{
					mapEntry("ActorsByRoleEntry", &descriptor.FieldDescriptorProto{
						Name:   proto.String("value"),
						Number: proto.Int32(2),
						Type:   descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
						Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					}),
					mapEntry("ItemsByIdEntry", &descriptor.FieldDescriptorProto{
						Name:     proto.String("value"),
						Number:   proto.Int32(2),
						Type:     descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						Label:    descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						TypeName: proto.String(".svc.Item"),
					}),
				},
				Field: []*descriptor.FieldDescriptorProto{
					{
						Name:    proto.String("actors"),
						Number:  proto.Int32(1),
						Type:    descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
						Label:   descriptor.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Options: withActorUUIDOption(t, true),
					},
					{
						Name:     proto.String("actors_by_role"),
						Number:   proto.Int32(2),
						Type:     descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						Label:    descriptor.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						TypeName: proto.String(".svc.Container.ActorsByRoleEntry"),
						Options:  withActorUUIDOption(t, true),
					},
					{
						Name:     proto.String("items"),
						Number:   proto.Int32(3),
						Type:     descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						Label:    descriptor.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						TypeName: proto.String(".svc.Item"),
					},
					{
						Name:     proto.String("items_by_id"),
						Number:   proto.Int32(4),
						Type:     descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						Label:    descriptor.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						TypeName: proto.String(".svc.Container.ItemsByIdEntry"),
					},
				},
			},
			{
				Name: proto.String("Resp"),
				Field: []*descriptor.FieldDescriptorProto{{
					Name:   proto.String("ok"),
					Number: proto.Int32(1),
					Type:   descriptor.FieldDescriptorProto_TYPE_BOOL.Enum(),
					Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				}},
			},
		},
		Service: []*descriptor.ServiceDescriptorProto{{
			Name: proto.String("S"),
			Method: []*descriptor.MethodDescriptorProto{{
				Name:       proto.String("M"),
				InputType:  proto.String(".svc.Container"),
				OutputType: proto.String(".svc.Resp"),
			}},
		}},
	}
}

// TestRunnerSilentlyIgnoresMisplacedAnnotations covers the genuinely
// ignored cases end-to-end: a request type whose only annotations sit on
// a non-string scalar (int64) and on a message-typed field whose subtree
// holds no string leaf must generate without errors AND emit no
// ActorUUID() accessor.
func TestRunnerSilentlyIgnoresMisplacedAnnotations(t *testing.T) {
	target := ignoredTargetFileDescriptor(t)
	req := &plugin_go.CodeGeneratorRequest{
		FileToGenerate: []string{"svc/foo.proto"},
		ProtoFile: []*descriptor.FileDescriptorProto{
			optionsFileDescriptor(),
			target,
		},
	}

	resp := Runner.Run(req)
	require.Nil(t, resp.Error, "plugin returned error: %v", resp.GetError())
	require.Len(t, resp.File, 1)

	out := resp.File[0].GetContent()
	assert.NotContains(t, out, "ActorUUID() []string",
		"misplaced annotations must not produce an accessor; output was:\n%s", out)
}

// optionsFileDescriptor returns a synthetic uber/auth/annotations/options.proto
// declaring the actor_uuid FieldOptions extension at the test field number.
func optionsFileDescriptor() *descriptor.FileDescriptorProto {
	return &descriptor.FileDescriptorProto{
		Name:    proto.String("uber/auth/annotations/options.proto"),
		Package: proto.String("uber.auth.annotations"),
		Syntax:  proto.String("proto3"),
		Extension: []*descriptor.FieldDescriptorProto{{
			Name:     proto.String("actor_uuid"),
			Number:   proto.Int32(_testActorUUIDFieldNumber),
			Type:     descriptor.FieldDescriptorProto_TYPE_BOOL.Enum(),
			Label:    descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			Extendee: proto.String(fieldOptionsExtendee),
		}},
		Options: &descriptor.FileOptions{
			GoPackage: proto.String("uber/auth/annotations/authpb"),
		},
	}
}

// targetFileDescriptor returns a synthetic svc/foo.proto containing a single
// service whose request type carries the actor_uuid annotation.
func targetFileDescriptor(t *testing.T) *descriptor.FileDescriptorProto {
	t.Helper()
	return &descriptor.FileDescriptorProto{
		Name:       proto.String("svc/foo.proto"),
		Package:    proto.String("svc"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"uber/auth/annotations/options.proto"},
		Options: &descriptor.FileOptions{
			GoPackage: proto.String("svc/foopb"),
		},
		MessageType: []*descriptor.DescriptorProto{
			{
				Name: proto.String("DeleteUserRequest"),
				Field: []*descriptor.FieldDescriptorProto{
					{
						Name:    proto.String("actor"),
						Number:  proto.Int32(1),
						Type:    descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
						Label:   descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Options: withActorUUIDOption(t, true),
					},
					{
						Name:   proto.String("user_id"),
						Number: proto.Int32(2),
						Type:   descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
						Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					},
				},
			},
			{
				Name: proto.String("DeleteUserResponse"),
				Field: []*descriptor.FieldDescriptorProto{{
					Name:   proto.String("ok"),
					Number: proto.Int32(1),
					Type:   descriptor.FieldDescriptorProto_TYPE_BOOL.Enum(),
					Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				}},
			},
		},
		Service: []*descriptor.ServiceDescriptorProto{{
			Name: proto.String("UserService"),
			Method: []*descriptor.MethodDescriptorProto{{
				Name:       proto.String("DeleteUser"),
				InputType:  proto.String(".svc.DeleteUserRequest"),
				OutputType: proto.String(".svc.DeleteUserResponse"),
			}},
		}},
	}
}

// nestedTargetFileDescriptor builds a synthetic svc/foo.proto whose
// service request type is Outer, which reaches the annotated leaf via
// a single message hop into Inner. Used by
// TestRunnerEmitsActorUUIDAccessorForNestedRequest.
func nestedTargetFileDescriptor(t *testing.T) *descriptor.FileDescriptorProto {
	t.Helper()
	return &descriptor.FileDescriptorProto{
		Name:       proto.String("svc/foo.proto"),
		Package:    proto.String("svc"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"uber/auth/annotations/options.proto"},
		Options: &descriptor.FileOptions{
			GoPackage: proto.String("svc/foopb"),
		},
		MessageType: []*descriptor.DescriptorProto{
			{
				Name: proto.String("Inner"),
				Field: []*descriptor.FieldDescriptorProto{{
					Name:    proto.String("uuid"),
					Number:  proto.Int32(1),
					Type:    descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
					Label:   descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Options: withActorUUIDOption(t, true),
				}},
			},
			{
				Name: proto.String("Outer"),
				Field: []*descriptor.FieldDescriptorProto{{
					Name:     proto.String("inner"),
					Number:   proto.Int32(1),
					Type:     descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					Label:    descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					TypeName: proto.String(".svc.Inner"),
				}},
			},
			{
				Name: proto.String("Resp"),
				Field: []*descriptor.FieldDescriptorProto{{
					Name:   proto.String("ok"),
					Number: proto.Int32(1),
					Type:   descriptor.FieldDescriptorProto_TYPE_BOOL.Enum(),
					Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				}},
			},
		},
		Service: []*descriptor.ServiceDescriptorProto{{
			Name: proto.String("S"),
			Method: []*descriptor.MethodDescriptorProto{{
				Name:       proto.String("M"),
				InputType:  proto.String(".svc.Outer"),
				OutputType: proto.String(".svc.Resp"),
			}},
		}},
	}
}

// ignoredTargetFileDescriptor builds a synthetic svc/foo.proto whose
// service request type is IgnoredReq with annotations only on genuinely
// ignored shapes: a non-string scalar (timestamp) and a message-typed
// field (creds) whose subtree holds no string leaf (its only field is an
// int64). No path should be found.
func ignoredTargetFileDescriptor(t *testing.T) *descriptor.FileDescriptorProto {
	t.Helper()
	return &descriptor.FileDescriptorProto{
		Name:       proto.String("svc/foo.proto"),
		Package:    proto.String("svc"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"uber/auth/annotations/options.proto"},
		Options: &descriptor.FileOptions{
			GoPackage: proto.String("svc/foopb"),
		},
		MessageType: []*descriptor.DescriptorProto{
			{
				Name: proto.String("Creds"),
				Field: []*descriptor.FieldDescriptorProto{{
					Name:    proto.String("issued_at"),
					Number:  proto.Int32(1),
					Type:    descriptor.FieldDescriptorProto_TYPE_INT64.Enum(),
					Label:   descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Options: withActorUUIDOption(t, true),
				}},
			},
			{
				Name: proto.String("IgnoredReq"),
				Field: []*descriptor.FieldDescriptorProto{
					{
						Name:    proto.String("timestamp"),
						Number:  proto.Int32(1),
						Type:    descriptor.FieldDescriptorProto_TYPE_INT64.Enum(),
						Label:   descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Options: withActorUUIDOption(t, true),
					},
					{
						Name:     proto.String("creds"),
						Number:   proto.Int32(2),
						Type:     descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						Label:    descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						TypeName: proto.String(".svc.Creds"),
						Options:  withActorUUIDOption(t, true),
					},
				},
			},
			{
				Name: proto.String("Resp"),
				Field: []*descriptor.FieldDescriptorProto{{
					Name:   proto.String("ok"),
					Number: proto.Int32(1),
					Type:   descriptor.FieldDescriptorProto_TYPE_BOOL.Enum(),
					Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				}},
			},
		},
		Service: []*descriptor.ServiceDescriptorProto{{
			Name: proto.String("S"),
			Method: []*descriptor.MethodDescriptorProto{{
				Name:       proto.String("M"),
				InputType:  proto.String(".svc.IgnoredReq"),
				OutputType: proto.String(".svc.Resp"),
			}},
		}},
	}
}
