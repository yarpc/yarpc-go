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

// End-to-end test for the (uber.auth.annotations.actor_uuid) annotation
// support in protoc-gen-yarpc-go. The fixtures in this package are
// generated from withuuid.proto by protoc-gen-yarpc-go using the same
// dynamic FQN-based discovery mechanism the production plugin uses, so
// these tests double as a contract that the generator continues to emit a
// usable ActorUUID() accessor on every annotated request type.
//
// Mirrors encoding/thrift/thriftrw-plugin-yarpc/uuid_test.go's
// TestGetGeneratedUUID, adapted to protobuf's getter conventions.

package withuuid

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActorUUIDAccessor(t *testing.T) {
	t.Run("DeleteUserRequest_returns_actor", func(t *testing.T) {
		req := &DeleteUserRequest{Actor: "alice", UserId: "u-123"}
		assert.Equal(t, []string{"alice"}, req.ActorUUID())
	})

	t.Run("GetUserRequest_returns_requester_id", func(t *testing.T) {
		req := &GetUserRequest{RequesterId: "bob", TargetId: "carol"}
		assert.Equal(t, []string{"bob"}, req.ActorUUID())
	})

	t.Run("ListUsersRequest_returns_caller_actor_id", func(t *testing.T) {
		req := &ListUsersRequest{CallerActorId: "dave", PageSize: 10}
		assert.Equal(t, []string{"dave"}, req.ActorUUID())
	})

	t.Run("nil_receiver_returns_zero_value", func(t *testing.T) {
		var req *DeleteUserRequest
		assert.Equal(t, []string{""}, req.ActorUUID())
	})

	t.Run("zero_value_returns_empty_string", func(t *testing.T) {
		assert.Equal(t, []string{""}, (&DeleteUserRequest{}).ActorUUID())
		assert.Equal(t, []string{""}, (&GetUserRequest{}).ActorUUID())
		assert.Equal(t, []string{""}, (&ListUsersRequest{}).ActorUUID())
	})
}

// TestActorUUIDAccessor_NestedRequest exercises the multi-step chain
// the generator emits when the annotated leaf sits behind several
// message hops. The accessor must walk the full chain and stay nil-safe
// when any intermediate hop is missing.
func TestActorUUIDAccessor_NestedRequest(t *testing.T) {
	t.Run("walks_three_hops_to_inner_uuid", func(t *testing.T) {
		req := &NestedRequest{
			Nested: &OuterLevel{
				Mid: &MidLevel{
					Inner: &InnerLevel{InnerUuid: "deep-actor"},
				},
			},
		}
		assert.Equal(t, []string{"deep-actor"}, req.ActorUUID())
	})

	t.Run("nil_intermediate_hop_is_nil_safe", func(t *testing.T) {
		req := &NestedRequest{Nested: &OuterLevel{}}
		assert.Equal(t, []string{""}, req.ActorUUID())
	})

	t.Run("zero_value_is_nil_safe_all_the_way", func(t *testing.T) {
		assert.Equal(t, []string{""}, (&NestedRequest{}).ActorUUID())
	})
}

// TestActorUUIDAccessor_CycleRequest covers the cycle-detection
// contract: a self-referencing message reached through a request type
// must not loop forever during code generation, and the generated
// chain must surface the annotated sibling.
func TestActorUUIDAccessor_CycleRequest(t *testing.T) {
	req := &CycleRequest{Root: &CycleNode{Id: "cycle-actor"}}
	assert.Equal(t, []string{"cycle-actor"}, req.ActorUUID())
}

// TestActorUUIDAccessor_AllAnnotationsCollected covers the "collect
// all" contract: a request type with several annotated fields yields
// one slice entry per field, in declaration order. Reaching more than
// one annotation is not an error.
func TestActorUUIDAccessor_AllAnnotationsCollected(t *testing.T) {
	t.Run("both_fields_set", func(t *testing.T) {
		req := &MultiAnnotatedRequest{FirstActor: "alice", SecondActor: "bob"}
		assert.Equal(t, []string{"alice", "bob"}, req.ActorUUID(),
			"each annotated field contributes its own entry, in declaration order")
	})

	t.Run("unset_field_keeps_its_slice_position", func(t *testing.T) {
		req := &MultiAnnotatedRequest{FirstActor: "alice"}
		assert.Equal(t, []string{"alice", ""}, req.ActorUUID(),
			"a missing field surfaces as the empty string but keeps its slice position")
	})
}

// TestActorUUIDAccessor_RepeatedString covers a `repeated string` leaf:
// the accessor appends every element, and an empty/unset slice
// contributes nothing.
func TestActorUUIDAccessor_RepeatedString(t *testing.T) {
	t.Run("all_elements_collected", func(t *testing.T) {
		req := &RepeatedActorsRequest{Actors: []string{"alice", "bob", "carol"}}
		assert.Equal(t, []string{"alice", "bob", "carol"}, req.ActorUUID(),
			"every element of the repeated string is surfaced, in order")
	})

	t.Run("empty_slice_contributes_nothing", func(t *testing.T) {
		assert.Empty(t, (&RepeatedActorsRequest{}).ActorUUID())
	})

	t.Run("nil_receiver_is_nil_safe", func(t *testing.T) {
		var req *RepeatedActorsRequest
		assert.Empty(t, req.ActorUUID())
	})
}

// TestActorUUIDAccessor_MapString covers a `map<string,string>` leaf: the
// accessor appends every value. Map iteration order is non-deterministic,
// so the values are compared without regard to order.
func TestActorUUIDAccessor_MapString(t *testing.T) {
	t.Run("all_values_collected", func(t *testing.T) {
		req := &MapActorsRequest{ActorsByRole: map[string]string{
			"admin":  "alice",
			"editor": "bob",
		}}
		assert.ElementsMatch(t, []string{"alice", "bob"}, req.ActorUUID(),
			"every value of the map is surfaced (order unspecified)")
	})

	t.Run("empty_map_contributes_nothing", func(t *testing.T) {
		assert.Empty(t, (&MapActorsRequest{}).ActorUUID())
	})
}

// TestActorUUIDAccessor_RepeatedMessage covers a `repeated Msg` hop: the
// accessor ranges over the slice and recurses into each element's
// annotated leaf.
func TestActorUUIDAccessor_RepeatedMessage(t *testing.T) {
	t.Run("each_element_recursed", func(t *testing.T) {
		req := &RepeatedMessageRequest{Items: []*ActorItem{
			{ActorUuid: "alice"},
			{ActorUuid: "bob"},
		}}
		assert.Equal(t, []string{"alice", "bob"}, req.ActorUUID())
	})

	t.Run("empty_slice_contributes_nothing", func(t *testing.T) {
		assert.Empty(t, (&RepeatedMessageRequest{}).ActorUUID())
	})

	t.Run("nil_element_is_nil_safe", func(t *testing.T) {
		req := &RepeatedMessageRequest{Items: []*ActorItem{nil, {ActorUuid: "bob"}}}
		assert.Equal(t, []string{"", "bob"}, req.ActorUUID(),
			"a nil element surfaces the zero value via the nil-safe getter")
	})
}

// TestActorUUIDAccessor_MapMessage covers a `map<string,Msg>` hop: the
// accessor ranges over the values and recurses into each one.
func TestActorUUIDAccessor_MapMessage(t *testing.T) {
	t.Run("each_value_recursed", func(t *testing.T) {
		req := &MapMessageRequest{ItemsById: map[string]*ActorItem{
			"a": {ActorUuid: "alice"},
			"b": {ActorUuid: "bob"},
		}}
		assert.ElementsMatch(t, []string{"alice", "bob"}, req.ActorUUID(),
			"every value of the map recurses to its annotated leaf (order unspecified)")
	})

	t.Run("empty_map_contributes_nothing", func(t *testing.T) {
		assert.Empty(t, (&MapMessageRequest{}).ActorUUID())
	})
}

// TestUnannotatedRequestHasNoActorUUID is the negative half of the
// contract: messages with no valid annotated leaf must not get an
// accessor. Asserted via reflection so the test fails to compile -> fails
// to even build the package - if a future regression starts emitting an
// accessor on a message that should be skipped.
func TestUnannotatedRequestHasNoActorUUID(t *testing.T) {
	cases := []struct {
		name string
		ptr  interface{}
	}{
		{"UnannotatedRequest", &UnannotatedRequest{}},
		{"DeleteUserResponse", &DeleteUserResponse{}},
		{"IgnoredAnnotationsRequest", &IgnoredAnnotationsRequest{}},
		{"IgnoredCredsContainer", &IgnoredCredsContainer{}},
		{"InnerLevel", &InnerLevel{}},
		{"MidLevel", &MidLevel{}},
		{"OuterLevel", &OuterLevel{}},
		{"CycleNode", &CycleNode{}},
		{"ActorItem", &ActorItem{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := reflect.TypeOf(tc.ptr).MethodByName("ActorUUID")
			assert.False(t, ok, "%s must not get an ActorUUID() accessor", tc.name)
		})
	}
}
