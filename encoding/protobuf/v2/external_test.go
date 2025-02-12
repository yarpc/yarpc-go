// Copyright (c) 2025 Uber Technologies, Inc.
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

package v2_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/x/restriction"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb/v2"
	"go.uber.org/yarpc/transport/grpc"
)

func TestFxClient(t *testing.T) {
	const serviceName = "foo-service"

	d := yarpc.NewDispatcher(yarpc.Config{
		Name: "foo-caller",
		Outbounds: yarpc.Outbounds{
			serviceName: {Unary: grpc.NewTransport().NewSingleOutbound("http://yarpc")},
		},
	})

	t.Run("success", func(t *testing.T) {
		assert.NotPanics(t, func() {
			p := testpb.FxTestYARPCClientParams{
				Provider: d,
			}
			f := testpb.NewFxTestYARPCClient(serviceName).(func(testpb.FxTestYARPCClientParams) testpb.FxTestYARPCClientResult)
			f(p)
		}, "failed to build client")
	})

	t.Run("invalid config", func(t *testing.T) {
		assert.PanicsWithValue(t, `no configured outbound transport for outbound key "nope"`, func() {
			p := testpb.FxTestYARPCClientParams{
				Provider: d,
			}
			f := testpb.NewFxTestYARPCClient("nope").(func(testpb.FxTestYARPCClientParams) testpb.FxTestYARPCClientResult)
			f(p)
		}, "expected panics")
	})

	t.Run("restriction success", func(t *testing.T) {
		r, err := restriction.NewChecker(restriction.Tuple{
			Transport: "grpc", Encoding: "proto",
		})
		require.NoError(t, err, "could not create restriction checker")

		assert.NotPanics(t, func() {
			p := testpb.FxTestYARPCClientParams{
				Provider:    d,
				Restriction: r,
			}
			f := testpb.NewFxTestYARPCClient(serviceName).(func(testpb.FxTestYARPCClientParams) testpb.FxTestYARPCClientResult)
			f(p)
		}, "failed to build client")
	})

	t.Run("restriction error", func(t *testing.T) {
		r, err := restriction.NewChecker(restriction.Tuple{
			Transport: "http", Encoding: "proto",
		})
		require.NoError(t, err, "could not create restriction checker")

		assert.PanicsWithValue(t, `"grpc/proto" is not a whitelisted combination, available: "http/proto"`, func() {
			p := testpb.FxTestYARPCClientParams{
				Provider:    d,
				Restriction: r,
			}
			f := testpb.NewFxTestYARPCClient(serviceName).(func(testpb.FxTestYARPCClientParams) testpb.FxTestYARPCClientResult)
			f(p)
		}, "failed to build client")
	})
}
