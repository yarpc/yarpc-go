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

package protobuf

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestActorUUIDValidatorFromOptions(t *testing.T) {
	t.Run("none installed returns nil", func(t *testing.T) {
		assert.Nil(t, ActorUUIDValidatorFromOptions(nil))
		assert.Nil(t, ActorUUIDValidatorFromOptions([]RegisterOption{}))
	})

	t.Run("returns the installed validator", func(t *testing.T) {
		var got []string
		validator := func(_ context.Context, uuids []string) error {
			got = uuids
			return nil
		}

		out := ActorUUIDValidatorFromOptions([]RegisterOption{WithActorUUIDValidator(validator)})
		require.NotNil(t, out, "the installed validator must be returned")

		require.NoError(t, out(context.Background(), []string{"alice"}))
		assert.Equal(t, []string{"alice"}, got, "the returned func must be the one we installed")
	})

	t.Run("last option wins", func(t *testing.T) {
		first := func(context.Context, []string) error { return errors.New("first") }
		second := func(context.Context, []string) error { return errors.New("second") }

		out := ActorUUIDValidatorFromOptions([]RegisterOption{
			WithActorUUIDValidator(first),
			WithActorUUIDValidator(second),
		})
		require.NotNil(t, out)
		assert.EqualError(t, out(context.Background(), nil), "second")
	})
}

func TestValidateActorUUID(t *testing.T) {
	t.Run("nil validator is a no-op", func(t *testing.T) {
		assert.NoError(t, ValidateActorUUID(context.Background(), nil, []string{"alice"}, "svc", "Method"))
	})

	t.Run("validator receives ctx and uuids and its nil result passes", func(t *testing.T) {
		type ctxKey struct{}
		ctx := context.WithValue(context.Background(), ctxKey{}, "v")

		var gotUUIDs []string
		var gotCtxValue interface{}
		validator := func(c context.Context, uuids []string) error {
			gotCtxValue = c.Value(ctxKey{})
			gotUUIDs = uuids
			return nil
		}

		require.NoError(t, ValidateActorUUID(ctx, validator, []string{"alice", "bob"}, "svc", "Method"))
		assert.Equal(t, []string{"alice", "bob"}, gotUUIDs)
		assert.Equal(t, "v", gotCtxValue, "the caller's context must be threaded through")
	})

	t.Run("rejection is wrapped as PermissionDenied naming service and procedure", func(t *testing.T) {
		denied := errors.New("validator denied")
		validator := func(context.Context, []string) error { return denied }

		err := ValidateActorUUID(context.Background(), validator, []string{"alice"}, "uber.example.UserService", "DeleteUser")
		require.Error(t, err)

		assert.Equal(t, yarpcerrors.CodePermissionDenied, yarpcerrors.FromError(err).Code())
		assert.ErrorIs(t, err, denied, "the original error must stay in the errors.Is chain")
		msg := err.Error()
		assert.Contains(t, msg, "uber.example.UserService")
		assert.Contains(t, msg, "DeleteUser")
	})
}
