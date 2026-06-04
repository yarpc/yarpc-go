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
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/protocol/binary"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/ptr"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/thrift"
	withservices "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/WITHSERVICES"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/WITHSERVICES/testserviceserver"
)

// testServiceImpl is a minimal hand-rolled implementation of
// testserviceserver.Interface that records what each method observed.
// It exists so the validator-gating tests below can prove the user
// handler was (or wasn't) reached without standing up a real
// dispatcher.
type testServiceImpl struct {
	called  bool
	gotCtx  context.Context
	gotArg  string
	gotUUID string
}

func (s *testServiceImpl) TestMethod(
	ctx context.Context,
	notInterested *string,
	interested *string,
) (string, error) {
	s.called = true
	s.gotCtx = ctx
	if interested != nil {
		s.gotArg = *interested
	}
	return "ok", nil
}

func (s *testServiceImpl) TestStructMethod(
	ctx context.Context,
	request *withservices.Struct,
) (string, error) {
	s.called = true
	s.gotCtx = ctx
	if request != nil && request.UserIdentifier != nil {
		s.gotUUID = *request.UserIdentifier
	}
	return "ok", nil
}

func (s *testServiceImpl) TestTypedefMethod(
	ctx context.Context,
	identifier *withservices.ActorIdentifier,
) (string, error) {
	s.called = true
	s.gotCtx = ctx
	if identifier != nil {
		s.gotUUID = string(*identifier)
	}
	return "ok", nil
}

// argsEncoder is the slice of the thriftrw-generated _Args API the
// drivers below need; both TestService_TestMethod_Args and the
// struct-/typedef-arg variants satisfy it.
type argsEncoder interface {
	Encode(stream.Writer) error
	ToWire() (wire.Value, error)
}

// encodeArgs marshals an _Args struct into the binary representation a
// real transport would deliver in transport.Request.Body. useStream
// picks between the wire.Value-encoded body (NoWire=false) and the
// streaming-binary body (default NoWire=true) so the codec matches
// what the handler the server was built with expects.
func encodeArgs(t *testing.T, args argsEncoder, useStream bool) io.Reader {
	t.Helper()
	var buf bytes.Buffer
	if useStream {
		sw := binary.Default.Writer(&buf)
		require.NoError(t, args.Encode(sw))
		require.NoError(t, sw.Close())
		return bytes.NewReader(buf.Bytes())
	}
	v, err := args.ToWire()
	require.NoError(t, err)
	require.NoError(t, binary.Default.Encode(v, &buf))
	return bytes.NewReader(buf.Bytes())
}

// buildRequest constructs a transport.Request that the procedure handler
// can dispatch on. The Procedure name is read off the procedure itself so
// the test doesn't bake in the "Service::method" format.
func buildRequest(procName string, body io.Reader) *transport.Request {
	return &transport.Request{
		Caller:    "caller-test",
		Service:   "callee-test",
		Procedure: procName,
		Encoding:  thrift.Encoding,
		Body:      body,
	}
}

// findProcedure picks the transport.Procedure for the given Thrift
// method name out of the slice testserviceserver.New returns. Now that
// TestService has multiple methods, tests can no longer assume a
// single-element slice.
func findProcedure(t *testing.T, procedures []transport.Procedure, methodName string) transport.Procedure {
	t.Helper()
	suffix := "::" + methodName
	for _, p := range procedures {
		if strings.HasSuffix(p.Name, suffix) {
			return p
		}
	}
	t.Fatalf("procedure for method %q not found", methodName)
	return transport.Procedure{}
}

// driveMethod is the shared core of the validator behaviour tests:
// build the server, encode args, run the procedure's unary handler, and
// hand the captured results back to the caller for assertions.
func driveMethod(
	t *testing.T,
	impl testserviceserver.Interface,
	methodName string,
	useStream bool,
	args argsEncoder,
	opts ...thrift.RegisterOption,
) (*transporttest.FakeResponseWriter, error) {
	t.Helper()
	procedures := testserviceserver.New(impl, opts...)
	proc := findProcedure(t, procedures, methodName)
	require.Equal(t, transport.Unary, proc.HandlerSpec.Type())

	rw := &transporttest.FakeResponseWriter{}
	req := buildRequest(proc.Name, encodeArgs(t, args, useStream))
	err := proc.HandlerSpec.Unary().Handle(context.Background(), req, rw)
	return rw, err
}

// driveTestMethod is the flat-arg specialisation kept for readability
// of the original tests, which all exercise testMethod's primitive args.
func driveTestMethod(
	t *testing.T,
	impl testserviceserver.Interface,
	useStream bool,
	args *withservices.TestService_TestMethod_Args,
	opts ...thrift.RegisterOption,
) (*transporttest.FakeResponseWriter, error) {
	t.Helper()
	return driveMethod(t, impl, "testMethod", useStream, args, opts...)
}

// TestActorUUIDValidator_NoValidatorBackwardCompat proves that a server
// constructed without WithActorUUIDValidator still runs the user handler:
// the generated `if h.actorUUIDValidator != nil` gate short-circuits
// when no validator was installed.
func TestActorUUIDValidator_NoValidatorBackwardCompat(t *testing.T) {
	for _, tc := range []struct {
		name      string
		useStream bool
		extraOpts []thrift.RegisterOption
	}{
		{name: "noWire", useStream: true, extraOpts: nil},
		{name: "wire", useStream: false, extraOpts: []thrift.RegisterOption{thrift.NoWire(false)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			impl := &testServiceImpl{}
			args := &withservices.TestService_TestMethod_Args{
				Interested: ptr.String("any-uuid"),
			}
			rw, err := driveTestMethod(t, impl, tc.useStream, args, tc.extraOpts...)
			require.NoError(t, err)
			assert.True(t, impl.called, "user handler must run when no validator is installed")
			assert.False(t, rw.IsApplicationError, "successful call should not be flagged as an application error")
		})
	}
}

// TestActorUUIDValidator_Allow proves that when the validator returns nil
// the generated code falls through to the user handler and threads the
// expected actorUUID through to the validator.
func TestActorUUIDValidator_Allow(t *testing.T) {
	for _, tc := range []struct {
		name      string
		useStream bool
		extraOpts []thrift.RegisterOption
	}{
		{name: "noWire", useStream: true, extraOpts: nil},
		{name: "wire", useStream: false, extraOpts: []thrift.RegisterOption{thrift.NoWire(false)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var seenUUID string
			validator := func(_ context.Context, uuid string) error {
				seenUUID = uuid
				return nil
			}

			impl := &testServiceImpl{}
			args := &withservices.TestService_TestMethod_Args{
				Interested: ptr.String("expected-uuid"),
			}
			opts := append([]thrift.RegisterOption{thrift.WithActorUUIDValidator(validator)}, tc.extraOpts...)

			_, err := driveTestMethod(t, impl, tc.useStream, args, opts...)
			require.NoError(t, err)
			assert.True(t, impl.called, "user handler must run when validator returns nil")
			assert.Equal(t, "expected-uuid", seenUUID, "validator should receive the decoded actorUUID")
			assert.Equal(t, "expected-uuid", impl.gotArg, "user handler should still see the original arg")
		})
	}
}

// TestActorUUIDValidator_Deny tests the validator returns an
// error the generated handler short-circuits: the user handler is never
// called and the validator's error reaches the caller wrapped in
// yarpcerrors.InvalidArgumentErrorf, with the original error preserved
// in the errors.Is chain (the generator wraps via %w).
func TestActorUUIDValidator_Deny(t *testing.T) {
	denied := errors.New("validator denied")

	for _, tc := range []struct {
		name      string
		useStream bool
		extraOpts []thrift.RegisterOption
	}{
		{name: "noWire", useStream: true, extraOpts: nil},
		{name: "wire", useStream: false, extraOpts: []thrift.RegisterOption{thrift.NoWire(false)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			validator := func(_ context.Context, _ string) error {
				return denied
			}

			impl := &testServiceImpl{}
			args := &withservices.TestService_TestMethod_Args{
				Interested: ptr.String("any-uuid"),
			}
			opts := append([]thrift.RegisterOption{thrift.WithActorUUIDValidator(validator)}, tc.extraOpts...)

			_, err := driveTestMethod(t, impl, tc.useStream, args, opts...)
			require.Error(t, err)
			assert.ErrorIs(t, err, denied, "validator error should remain in the wrapped errors.Is chain")
			assert.False(t, impl.called, "user handler must NOT run when validator rejects")
		})
	}
}

// TestActorUUIDValidator_EmptyActorUUID proves the validator still fires
// when the annotated arg is nil/empty - the generated GetInterested()
// returns "" and that's what the validator sees. Policy on whether empty
// is acceptable belongs to the validator, not the generated code.
func TestActorUUIDValidator_EmptyActorUUID(t *testing.T) {
	var seenUUID string
	var calls int
	validator := func(_ context.Context, uuid string) error {
		calls++
		seenUUID = uuid
		return nil
	}

	impl := &testServiceImpl{}
	args := &withservices.TestService_TestMethod_Args{} // Interested == nil
	_, err := driveTestMethod(t, impl, true, args, thrift.WithActorUUIDValidator(validator))
	require.NoError(t, err)
	assert.Equal(t, 1, calls, "validator should still fire even with empty actorUUID")
	assert.Equal(t, "", seenUUID, "empty optional arg should surface as the empty string")
	assert.True(t, impl.called, "validator returning nil should let the handler run")
}

// TestActorUUIDValidator_StructArg proves the validator fires for a
// struct-typed arg whose own field is annotated and sees the same
// UUID the user handler reads from the inner field. This is the
// regression check for the "annotation lives on a field of a struct
// passed as a method arg" bug.
func TestActorUUIDValidator_StructArg(t *testing.T) {
	var seenUUID string
	validator := func(_ context.Context, uuid string) error {
		seenUUID = uuid
		return nil
	}

	impl := &testServiceImpl{}
	args := &withservices.TestService_TestStructMethod_Args{
		Request: &withservices.Struct{
			UserIdentifier: ptr.String("expected-struct-uuid"),
		},
	}
	_, err := driveMethod(t, impl, "testStructMethod", true, args,
		thrift.WithActorUUIDValidator(validator))
	require.NoError(t, err)
	assert.True(t, impl.called)
	assert.Equal(t, "expected-struct-uuid", seenUUID,
		"validator should receive the UUID chained through the struct arg")
	assert.Equal(t, "expected-struct-uuid", impl.gotUUID)
}

// TestActorUUIDValidator_TypedefArg proves the validator fires for an
// arg whose Thrift type is a typedef of string. The string(...) cast
// the generator emits is what makes this compile end to end; this
// test pins the runtime behaviour.
func TestActorUUIDValidator_TypedefArg(t *testing.T) {
	var seenUUID string
	validator := func(_ context.Context, uuid string) error {
		seenUUID = uuid
		return nil
	}

	impl := &testServiceImpl{}
	id := withservices.ActorIdentifier("expected-typedef-uuid")
	args := &withservices.TestService_TestTypedefMethod_Args{Identifier: &id}
	_, err := driveMethod(t, impl, "testTypedefMethod", true, args,
		thrift.WithActorUUIDValidator(validator))
	require.NoError(t, err)
	assert.True(t, impl.called)
	assert.Equal(t, "expected-typedef-uuid", seenUUID)
	assert.Equal(t, "expected-typedef-uuid", impl.gotUUID)
}
