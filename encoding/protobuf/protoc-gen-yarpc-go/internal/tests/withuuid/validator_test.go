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

// End-to-end test for the ActorUUID validator wiring protoc-gen-yarpc-go
// injects into generated servers. Mirrors
// encoding/thrift/thriftrw-plugin-yarpc/validator_test.go, adapted to
// protobuf's generated handlers and []string accessor.

package withuuid

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/protobuf"
	"go.uber.org/yarpc/yarpcerrors"
)

// recordingServer is a minimal UserServiceYARPCServer that records
// whether the unary handler under test was reached and what actor it
// observed. Only the methods exercised below carry behaviour; the rest
// satisfy the interface.
type recordingServer struct {
	called    bool
	gotActor  string
	gotCaller string
}

func (s *recordingServer) DeleteUser(_ context.Context, req *DeleteUserRequest) (*DeleteUserResponse, error) {
	s.called = true
	s.gotActor = req.GetActor()
	return &DeleteUserResponse{Ok: true}, nil
}

func (s *recordingServer) GetUser(context.Context, *GetUserRequest) (*GetUserResponse, error) {
	return &GetUserResponse{}, nil
}
func (s *recordingServer) Ping(context.Context, *UnannotatedRequest) (*UnannotatedResponse, error) {
	s.called = true
	return &UnannotatedResponse{Ok: true}, nil
}
func (s *recordingServer) CredentialedAction(context.Context, *NestedRequest) (*DeleteUserResponse, error) {
	return &DeleteUserResponse{}, nil
}
func (s *recordingServer) CycleAction(context.Context, *CycleRequest) (*DeleteUserResponse, error) {
	return &DeleteUserResponse{}, nil
}
func (s *recordingServer) MultipleAction(context.Context, *MultiAnnotatedRequest) (*DeleteUserResponse, error) {
	return &DeleteUserResponse{}, nil
}
func (s *recordingServer) RepeatedActorsAction(context.Context, *RepeatedActorsRequest) (*DeleteUserResponse, error) {
	return &DeleteUserResponse{}, nil
}
func (s *recordingServer) MapActorsAction(context.Context, *MapActorsRequest) (*DeleteUserResponse, error) {
	return &DeleteUserResponse{}, nil
}
func (s *recordingServer) RepeatedMessageAction(context.Context, *RepeatedMessageRequest) (*DeleteUserResponse, error) {
	return &DeleteUserResponse{}, nil
}
func (s *recordingServer) MapMessageAction(context.Context, *MapMessageRequest) (*DeleteUserResponse, error) {
	return &DeleteUserResponse{}, nil
}
func (s *recordingServer) IgnoredAction(context.Context, *IgnoredAnnotationsRequest) (*DeleteUserResponse, error) {
	return &DeleteUserResponse{}, nil
}
func (s *recordingServer) ListUsers(*ListUsersRequest, UserServiceServiceListUsersYARPCServer) error {
	return nil
}

// unaryProcedure finds the proto-encoding unary procedure for the given
// method name out of the slice BuildUserServiceYARPCProcedures returns
// (each method emits both a proto and a JSON procedure).
func unaryProcedure(t *testing.T, procedures []transport.Procedure, methodName string) transport.Procedure {
	t.Helper()
	for _, p := range procedures {
		if p.Encoding != protobuf.Encoding || p.HandlerSpec.Type() != transport.Unary {
			continue
		}
		if bytes.HasSuffix([]byte(p.Name), []byte(methodName)) {
			return p
		}
	}
	t.Fatalf("unary proto procedure for method %q not found", methodName)
	return transport.Procedure{}
}

// driveDeleteUser builds a server with the given options, marshals req,
// and runs the DeleteUser unary handler, returning any handler error.
func driveDeleteUser(t *testing.T, impl UserServiceYARPCServer, req *DeleteUserRequest, opts ...protobuf.RegisterOption) error {
	t.Helper()
	procedures := BuildUserServiceYARPCProcedures(impl, opts...)
	proc := unaryProcedure(t, procedures, "DeleteUser")

	body, err := proto.Marshal(req)
	require.NoError(t, err)

	return proc.HandlerSpec.Unary().Handle(
		context.Background(),
		&transport.Request{
			Caller:    "caller-test",
			Service:   "callee-test",
			Procedure: proc.Name,
			Encoding:  protobuf.Encoding,
			Body:      bytes.NewReader(body),
		},
		&transporttest.FakeResponseWriter{},
	)
}

// TestActorUUIDValidator_NoValidatorBackwardCompat proves a server built
// without WithActorUUIDValidator still runs the user handler: the
// generated nil-guard short-circuits when no validator is installed.
func TestActorUUIDValidator_NoValidatorBackwardCompat(t *testing.T) {
	impl := &recordingServer{}
	err := driveDeleteUser(t, impl, &DeleteUserRequest{Actor: "alice"})
	require.NoError(t, err)
	assert.True(t, impl.called, "user handler must run when no validator is installed")
}

// TestActorUUIDValidator_Allow proves that when the validator returns nil
// the generated code falls through to the user handler and threads the
// decoded actor UUID slice through to the validator.
func TestActorUUIDValidator_Allow(t *testing.T) {
	var seen []string
	validator := func(_ context.Context, actorUUIDs []string) error {
		seen = actorUUIDs
		return nil
	}

	impl := &recordingServer{}
	err := driveDeleteUser(t, impl, &DeleteUserRequest{Actor: "alice"},
		protobuf.WithActorUUIDValidator(validator))
	require.NoError(t, err)
	assert.True(t, impl.called, "user handler must run when validator returns nil")
	assert.Equal(t, []string{"alice"}, seen, "validator should receive the decoded actor UUIDs")
	assert.Equal(t, "alice", impl.gotActor, "user handler should still see the original request")
}

// TestActorUUIDValidator_Deny proves that when the validator returns an
// error the generated handler short-circuits: the user handler is never
// called and the validator's error reaches the caller wrapped in an
// InvalidArgument YARPC error, with the original error preserved in the
// errors.Is chain (the generator wraps via %w).
func TestActorUUIDValidator_Deny(t *testing.T) {
	denied := errors.New("validator denied")
	validator := func(context.Context, []string) error { return denied }

	impl := &recordingServer{}
	err := driveDeleteUser(t, impl, &DeleteUserRequest{Actor: "alice"},
		protobuf.WithActorUUIDValidator(validator))
	require.Error(t, err)
	assert.ErrorIs(t, err, denied, "validator error should remain in the wrapped errors.Is chain")
	assert.Equal(t, yarpcerrors.CodeInvalidArgument, yarpcerrors.FromError(err).Code(),
		"a rejected actor UUID should surface as InvalidArgument")
	assert.False(t, impl.called, "user handler must NOT run when validator rejects")
}

// TestActorUUIDValidator_EmptyActorUUID proves the validator still fires
// when the annotated field is unset: the generated accessor returns
// []string{""} and that is what the validator sees. Policy on whether
// empty is acceptable belongs to the validator, not the generated code.
func TestActorUUIDValidator_EmptyActorUUID(t *testing.T) {
	var calls int
	var seen []string
	validator := func(_ context.Context, actorUUIDs []string) error {
		calls++
		seen = actorUUIDs
		return nil
	}

	impl := &recordingServer{}
	err := driveDeleteUser(t, impl, &DeleteUserRequest{}, protobuf.WithActorUUIDValidator(validator))
	require.NoError(t, err)
	assert.Equal(t, 1, calls, "validator should still fire even with an empty actor UUID")
	assert.Equal(t, []string{""}, seen, "an unset annotated field surfaces as the empty string")
	assert.True(t, impl.called, "validator returning nil should let the handler run")
}

// TestActorUUIDValidator_UnannotatedMethodSkipsValidator proves the
// generated code never calls the validator for a method whose request
// type carries no annotation (Ping / UnannotatedRequest): such handlers
// have no ActorUUID() accessor, so the validator gate is omitted
// entirely and the handler runs even when a validator is installed.
func TestActorUUIDValidator_UnannotatedMethodSkipsValidator(t *testing.T) {
	var calls int
	validator := func(context.Context, []string) error { calls++; return errors.New("should not run") }

	impl := &recordingServer{}
	procedures := BuildUserServiceYARPCProcedures(impl, protobuf.WithActorUUIDValidator(validator))
	proc := unaryProcedure(t, procedures, "Ping")

	body, err := proto.Marshal(&UnannotatedRequest{Token: "t"})
	require.NoError(t, err)
	err = proc.HandlerSpec.Unary().Handle(
		context.Background(),
		&transport.Request{
			Procedure: proc.Name,
			Encoding:  protobuf.Encoding,
			Body:      bytes.NewReader(body),
		},
		&transporttest.FakeResponseWriter{},
	)
	require.NoError(t, err)
	assert.Equal(t, 0, calls, "validator must not fire for an unannotated method")
	assert.True(t, impl.called, "the unannotated method's handler must still run")
}
