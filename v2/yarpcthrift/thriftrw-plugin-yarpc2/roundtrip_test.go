// Copyright (c) 2018 Uber Technologies, Inc.
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

package main_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"testing"

	"go.uber.org/yarpc/v2/yarpcerror"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/ptr"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internaltesttime"
	"go.uber.org/yarpc/v2/yarpchttp"
	"go.uber.org/yarpc/v2/yarpcrouter"
	"go.uber.org/yarpc/v2/yarpcthrift"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/atomic"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/atomic/readonlystoreclient"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/atomic/readonlystoreserver"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/atomic/storeclient"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/atomic/storeserver"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/common/baseserviceclient"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/common/baseserviceserver"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/common/emptyserviceclient"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/common/emptyserviceserver"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/common/extendemptyclient"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/common/extendemptyserver"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/common/extendonlyclient"
	"go.uber.org/yarpc/v2/yarpcthrift/thriftrw-plugin-yarpc2/internal/tests/common/extendonlyserver"
)

func TestRoundTrip(t *testing.T) {
	tests := []struct{ enveloped, multiplexed bool }{
		{true, true},
		{true, false},
		{false, true},
		{false, false},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("enveloped(%v)/multiplexed(%v)", tt.enveloped, tt.multiplexed)
		t.Run(name, func(t *testing.T) { testRoundTrip(t, tt.enveloped, tt.multiplexed) })
	}
}

func testRoundTrip(t *testing.T, enveloped, multiplexed bool) {
	var serverOpts []yarpcthrift.RegisterOption
	if enveloped {
		serverOpts = append(serverOpts, yarpcthrift.Enveloped)
	}

	var clientOpts []yarpcthrift.ClientOption
	if enveloped {
		clientOpts = append(clientOpts, yarpcthrift.Enveloped)
	}
	if multiplexed {
		clientOpts = append(clientOpts, yarpcthrift.Multiplexed)
	}

	tests := []struct {
		desc          string
		procedures    []yarpc.EncodingProcedure
		newClientFunc interface{}

		// if method is non-empty, client.method(ctx, methodArgs...) will be called
		method     string
		methodArgs []interface{}

		wantResult interface{}
		wantError  error
	}{
		{
			desc:          "empty service",
			procedures:    emptyserviceserver.New(struct{}{}, serverOpts...),
			newClientFunc: emptyserviceclient.New,
		},
		{
			desc:          "extend empty: hello",
			procedures:    extendemptyserver.New(extendEmptyHandler{}, serverOpts...),
			newClientFunc: extendemptyclient.New,
			method:        "Hello",
		},
		{
			desc:          "base: healthy",
			procedures:    baseserviceserver.New(extendEmptyHandler{}, serverOpts...),
			newClientFunc: baseserviceclient.New,
			method:        "Healthy",
			wantResult:    true,
		},
		{
			desc:          "extend only: healthy",
			procedures:    extendonlyserver.New(&storeHandler{healthy: true}, serverOpts...),
			newClientFunc: extendonlyclient.New,
			method:        "Healthy",
			wantResult:    true,
		},
		{
			desc:          "store: healthy",
			procedures:    storeserver.New(&storeHandler{healthy: true}, serverOpts...),
			newClientFunc: storeclient.New,
			method:        "Healthy",
			wantResult:    true,
		},
		{
			desc:          "store: healthy with base client",
			procedures:    storeserver.New(&storeHandler{healthy: true}, serverOpts...),
			newClientFunc: baseserviceclient.New,
			method:        "Healthy",
			wantResult:    true,
		},
		{
			desc:          "store: unhealthy",
			procedures:    storeserver.New(&storeHandler{}, serverOpts...),
			newClientFunc: storeclient.New,
			method:        "Healthy",
			wantResult:    false,
		},
		{
			desc:          "store: increment",
			procedures:    storeserver.New(&storeHandler{}, serverOpts...),
			newClientFunc: storeclient.New,
			method:        "Increment",
			methodArgs:    []interface{}{ptr.String("foo"), ptr.Int64(42)},
		},
		{
			desc:          "store: compare and swap",
			procedures:    storeserver.New(&storeHandler{}, serverOpts...),
			newClientFunc: storeclient.New,
			method:        "CompareAndSwap",
			methodArgs: []interface{}{
				&atomic.CompareAndSwap{
					Key:          "foo",
					CurrentValue: 42,
					NewValue:     420,
				},
			},
		},
		{
			desc: "store: compare and swap failure",
			procedures: storeserver.New(&storeHandler{
				failWith: &atomic.IntegerMismatchError{
					ExpectedValue: 42,
					GotValue:      43,
				},
			}, serverOpts...),
			newClientFunc: storeclient.New,
			method:        "CompareAndSwap",
			methodArgs: []interface{}{
				&atomic.CompareAndSwap{
					Key:          "foo",
					CurrentValue: 42,
					NewValue:     420,
				},
			},
			// TODO(mhp): After we have a way to map errors between YARPC errors
			// and thrift exceptions, this should be revisited so that the check
			// below actually returns well-defined thrift exceptions.
			wantError: yarpcerror.WrapHandlerError(&atomic.IntegerMismatchError{
				ExpectedValue: 42,
				GotValue:      43,
			}, "roundtrip-server", "Store::compareAndSwap"),
		},
		{
			desc:          "store: integer with readonly client",
			procedures:    storeserver.New(&storeHandler{integer: 42}, serverOpts...),
			newClientFunc: readonlystoreclient.New,
			method:        "Integer",
			methodArgs:    []interface{}{ptr.String("foo")},
			wantResult:    int64(42),
		},
		{
			desc: "readonly store: integer error with rw client",
			procedures: readonlystoreserver.New(&storeHandler{
				failWith: &atomic.KeyDoesNotExist{Key: ptr.String("foo")},
			}, serverOpts...),
			newClientFunc: storeclient.New,
			method:        "Integer",
			methodArgs:    []interface{}{ptr.String("foo")},
			// TODO(mhp): After we have a way to map errors between YARPC errors
			// and thrift exceptions, this should be revisited so that the check
			// below actually returns well-defined thrift exceptions.
			wantError: yarpcerror.WrapHandlerError(
				&atomic.KeyDoesNotExist{Key: ptr.String("foo")},
				"roundtrip-server",
				"ReadOnlyStore::integer",
			),
		},
		{
			desc:          "readonly store: integer with readonly client",
			procedures:    readonlystoreserver.New(&storeHandler{integer: 42}, serverOpts...),
			newClientFunc: readonlystoreclient.New,
			method:        "Integer",
			methodArgs:    []interface{}{ptr.String("foo")},
			wantResult:    int64(42),
		},
		{
			desc:          "readonly store: integer with rw client",
			procedures:    readonlystoreserver.New(&storeHandler{integer: 42}, serverOpts...),
			newClientFunc: storeclient.New,
			method:        "Integer",
			methodArgs:    []interface{}{ptr.String("foo")},
			wantResult:    int64(42),
		},
		{
			desc: "readonly store: integer failure with rw client",
			procedures: readonlystoreserver.New(&storeHandler{
				failWith: &atomic.KeyDoesNotExist{Key: ptr.String("foo")},
			}, serverOpts...),
			newClientFunc: storeclient.New,
			method:        "Integer",
			methodArgs:    []interface{}{ptr.String("foo")},
			// TODO(mhp): After we have a way to map errors between YARPC errors
			// and thrift exceptions, this should be revisited so that the check
			// below actually returns well-defined thrift exceptions.
			wantError: yarpcerror.WrapHandlerError(
				&atomic.KeyDoesNotExist{Key: ptr.String("foo")},
				"roundtrip-server",
				"ReadOnlyStore::integer",
			),
		},
		{
			desc: "store: integer failure",
			procedures: storeserver.New(&storeHandler{
				failWith: &atomic.KeyDoesNotExist{Key: ptr.String("foo")},
			}, serverOpts...),
			newClientFunc: storeclient.New,
			method:        "Integer",
			methodArgs:    []interface{}{ptr.String("foo")},
			// TODO(mhp): After we have a way to map errors between YARPC errors
			// and thrift exceptions, this should be revisited so that the check
			// below actually returns well-defined thrift exceptions.
			wantError: yarpcerror.WrapHandlerError(
				&atomic.KeyDoesNotExist{Key: ptr.String("foo")},
				"roundtrip-server",
				"ReadOnlyStore::integer",
			),
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			procedures, err := yarpc.EncodingToTransportProcedures(tt.procedures)
			require.NoError(t, err)
			router := yarpcrouter.NewMapRouter("roundtrip-server", procedures)
			listener, err := net.Listen("tcp", ":0")
			require.NoError(t, err)
			inbound := &yarpchttp.Inbound{
				Listener: listener,
				Router:   router,
			}
			require.NoError(t, inbound.Start(context.Background()), "failed to start server")
			defer func() {
				assert.NoError(t, inbound.Stop(context.Background()), "failed to stop server")
			}()

			dialer := &yarpchttp.Dialer{}
			require.NoError(t, dialer.Start(context.Background()))
			outbound := &yarpchttp.Outbound{
				URL:    &url.URL{Scheme: "http", Host: listener.Addr().String()},
				Dialer: dialer,
			}
			yarpcClient := yarpc.Client{
				Caller:  "roundtrip-client",
				Service: "roundtrip-server",
				Unary:   outbound,
			}

			// Verify that newClientFunc was valid
			newClientFuncType := reflect.TypeOf(tt.newClientFunc)
			require.Equal(t, reflect.Func, newClientFuncType.Kind(),
				"invalid test: newClientFunc must be a function")
			require.Equal(t, 1, newClientFuncType.NumOut(),
				"invalid test: newClientFunc must return a single result")

			clientType := newClientFuncType.Out(0)
			require.Equal(t, reflect.Interface, clientType.Kind(),
				"invalid test: newClientFunc must return an Interface")

			clientArgs := []reflect.Value{reflect.ValueOf(yarpcClient)}
			for _, opt := range clientOpts {
				clientArgs = append(clientArgs, reflect.ValueOf(opt))
			}

			client := reflect.ValueOf(tt.newClientFunc).
				Call(clientArgs)[0]

			if tt.method == "" {
				return
			}

			// Equivalent to,
			//
			// 	... := client.$method(ctx, $methodArgs...)
			method := client.MethodByName(tt.method)
			assert.True(t, method.IsValid(), "Method %q not found", tt.method)

			ctx, cancel := context.WithTimeout(ctx, 200*internaltesttime.Millisecond)
			defer cancel()

			args := append([]interface{}{ctx}, tt.methodArgs...)
			returns := method.Call(values(args...))

			switch len(returns) {
			case 1: // error
				err, _ := returns[0].Interface().(error)
				assert.Equal(t, tt.wantError, err)
			case 2: // (result, err)
				result := returns[0].Interface()
				err, _ := returns[1].Interface().(error)
				if tt.wantError != nil {
					assert.Equal(t, tt.wantError, err)
				} else {
					require.NoError(t, err, "expected success")
					assert.Equal(t, tt.wantResult, result)
				}
			default:
				t.Fatalf(
					"impossible: %q returned %d results; only up to 2 are allowed", tt.method, len(returns))
			}
		})
	}
}

func values(xs ...interface{}) []reflect.Value {
	vs := make([]reflect.Value, len(xs))
	for i, x := range xs {
		vs[i] = reflect.ValueOf(x)
	}
	return vs
}

type storeHandler struct {
	healthy  bool
	failWith error
	integer  int64
}

func (h *storeHandler) Healthy(ctx context.Context) (bool, error) {
	return h.healthy, h.failWith
}

func (h *storeHandler) CompareAndSwap(ctx context.Context, req *atomic.CompareAndSwap) error {
	return h.failWith
}

func (h *storeHandler) Forget(ctx context.Context, key *string) error {
	return h.failWith
}

func (h *storeHandler) Increment(ctx context.Context, key *string, value *int64) error {
	return h.failWith
}

func (h *storeHandler) Integer(ctx context.Context, key *string) (int64, error) {
	return h.integer, h.failWith
}

type extendEmptyHandler struct{}

func (extendEmptyHandler) Hello(ctx context.Context) error {
	return nil
}

func (extendEmptyHandler) Healthy(ctx context.Context) (bool, error) {
	return true, nil
}
