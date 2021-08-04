// Copyright (c) 2021 Uber Technologies, Inc.
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
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/ptr"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/readonlystoreclient"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/readonlystoreserver"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/storeclient"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/storeserver"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/baseserviceclient"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/baseserviceserver"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/emptyserviceclient"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/emptyserviceserver"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/extendemptyclient"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/extendemptyserver"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/extendonlyclient"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/extendonlyserver"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestRoundTrip(t *testing.T) {
	tests := []struct{ enveloped, multiplexed, nowire bool }{
		{true, true, true},
		{true, true, false},
		// Skipping for now until flaky test fixed.
		// Uncomment this when fixed.
		// https://github.com/yarpc/yarpc-go/issues/1171
		//{true, false, true},
		//{true, false, false},
		{false, true, true},
		{false, true, false},
		{false, false, true},
		{false, false, false},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("enveloped(%v)/multiplexed(%v)/nowire(%v)", tt.enveloped, tt.multiplexed, tt.nowire)
		t.Run(name, func(t *testing.T) { testRoundTrip(t, tt.enveloped, tt.multiplexed, tt.nowire) })
	}
}

func testRoundTrip(t *testing.T, enveloped, multiplexed, nowire bool) {
	t.Helper()

	var serverOpts []thrift.RegisterOption
	if enveloped {
		serverOpts = append(serverOpts, thrift.Enveloped)
	}
	serverOpts = append(serverOpts, thrift.NoWire(nowire))

	var clientOpts []string
	if enveloped {
		clientOpts = append(clientOpts, "enveloped")
	}
	if multiplexed {
		clientOpts = append(clientOpts, "multiplexed")
	}

	var thriftTag string
	if len(clientOpts) > 0 {
		thriftTag = fmt.Sprintf(` thrift:"%v"`, strings.Join(clientOpts, ","))
	}

	tests := []struct {
		desc          string
		procedures    []transport.Procedure
		newClientFunc interface{}

		// if method is non-empty, client.method(ctx, methodArgs...) will be called
		method     string
		methodArgs []interface{}

		wantAck    bool
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
			wantError: &atomic.IntegerMismatchError{
				ExpectedValue: 42,
				GotValue:      43,
			},
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
			desc: "store: integer failure",
			procedures: storeserver.New(&storeHandler{
				failWith: &atomic.KeyDoesNotExist{Key: ptr.String("foo")},
			}, serverOpts...),
			newClientFunc: storeclient.New,
			method:        "Integer",
			methodArgs:    []interface{}{ptr.String("foo")},
			wantError:     &atomic.KeyDoesNotExist{Key: ptr.String("foo")},
		},
		{
			desc:          "store: forget",
			procedures:    storeserver.New(&storeHandler{}, serverOpts...),
			newClientFunc: storeclient.New,
			method:        "Forget",
			methodArgs:    []interface{}{ptr.String("foo")},
			wantAck:       true,
		},
		{
			desc:          "store: forget error",
			procedures:    storeserver.New(&storeHandler{failWith: errors.New("great sadness")}, serverOpts...),
			newClientFunc: storeclient.New,
			method:        "Forget",
			methodArgs:    []interface{}{ptr.String("foo")},
			wantAck:       true,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			httpInbound := http.NewTransport().NewInbound("127.0.0.1:0")

			server := yarpc.NewDispatcher(yarpc.Config{
				Name:     "roundtrip-server",
				Inbounds: yarpc.Inbounds{httpInbound},
			})
			server.Register(tt.procedures)
			require.NoError(t, server.Start())
			defer server.Stop()

			outbound := http.NewTransport().NewSingleOutbound(
				fmt.Sprintf("http://%v", yarpctest.ZeroAddrToHostPort(httpInbound.Addr())))

			dispatcher := yarpc.NewDispatcher(yarpc.Config{
				Name: "roundtrip-client",
				Outbounds: yarpc.Outbounds{
					"roundtrip-server": {
						Unary:  outbound,
						Oneway: outbound,
					},
				},
			})
			require.NoError(t, dispatcher.Start())
			defer dispatcher.Stop()

			// Verify that newClientFunc was valid
			newClientFuncType := reflect.TypeOf(tt.newClientFunc)
			require.Equal(t, reflect.Func, newClientFuncType.Kind(),
				"invalid test: newClientFunc must be a function")
			require.Equal(t, 1, newClientFuncType.NumOut(),
				"invalid test: newClientFunc must return a single result")

			clientType := newClientFuncType.Out(0)
			require.Equal(t, reflect.Interface, clientType.Kind(),
				"invalid test: newClientFunc must return an Interface")

			// The following blob is equivalent to,
			//
			// 	var clientHolder struct {
			// 		Client ${service}client.Interface `service:"roundtrip-server"`
			// 	}
			// 	yarpc.InjectClients(dispatcher, &clientHolder)
			// 	client := clientHolder.Client
			//
			// Optionally with the `thrift:"..."` tag if thriftTag was
			// non-empty.
			structType := reflect.StructOf([]reflect.StructField{
				{
					Name: "Client",
					Type: clientType,
					Tag:  reflect.StructTag(`service:"roundtrip-server"` + thriftTag),
				},
			})
			clientHolder := reflect.New(structType).Elem()
			yarpc.InjectClients(dispatcher, clientHolder.Addr().Interface())
			client := clientHolder.Field(0)
			assert.NotNil(t, client.Interface(), "InjectClients did not provide a client")

			if tt.method == "" {
				return
			}

			// Equivalent to,
			//
			// 	... := client.$method(ctx, $methodArgs...)
			method := client.MethodByName(tt.method)
			assert.True(t, method.IsValid(), "Method %q not found", tt.method)

			ctx, cancel := context.WithTimeout(ctx, 200*testtime.Millisecond)
			defer cancel()

			args := append([]interface{}{ctx}, tt.methodArgs...)
			returns := method.Call(values(args...))

			switch len(returns) {
			case 1: // error
				err, _ := returns[0].Interface().(error)
				assert.Equal(t, tt.wantError, err)
			case 2: // (ack/result, err)
				result := returns[0].Interface()
				err, _ := returns[1].Interface().(error)
				if tt.wantError != nil {
					assert.Equal(t, tt.wantError, err)
				} else {
					if !assert.NoError(t, err, "expected success") {
						return
					}

					if tt.wantAck {
						assert.Implements(t, (*transport.Ack)(nil), result, "expected a non-nil ack")
						assert.NotNil(t, result, "expected a non-nil ack")
					} else {
						assert.Equal(t, tt.wantResult, result)
					}
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

func TestFromWireInvalidArg(t *testing.T) {
	procedures := storeserver.New(nil /*server impl*/)
	require.Len(t, procedures, 5, "unexpected number of procedures")

	procedure := procedures[2]
	require.Equal(t, "Store::compareAndSwap", procedure.Name)
	require.Equal(t, transport.Unary, procedure.HandlerSpec.Type())

	// This struct is identical to the `CompareAndSwap` wrapper
	// `Store_CompareAndSwap_Args`, except all fields are optional. This will
	// allow us to produce an invalid payload.
	//
	// The handler will fail to unmarshal this type as it is missing required
	// fields.
	request := &atomic.OptionalCompareAndSwapWrapper{Cas: &atomic.OptionalCompareAndSwap{}}
	val, err := request.ToWire()
	require.NoError(t, err, "unable to covert type to wire.Value")

	var body bytes.Buffer
	err = (protocol.Binary).Encode(val, &body)
	require.NoError(t, err, "could not marshal message to bytes")

	err = procedure.HandlerSpec.Unary().Handle(context.Background(), &transport.Request{
		Encoding: thrift.Encoding,
		Body:     &body,
	}, nil /*response writer*/)

	require.Error(t, err, "expected handler error")
	assert.True(t, yarpcerrors.IsStatus(err), "unkown error")
	assert.Equal(t, yarpcerrors.CodeInvalidArgument, yarpcerrors.FromError(err).Code(), "unexpected code")
	assert.Contains(t, yarpcerrors.FromError(err).Message(), "Store", "expected Thrift service name in message")
	assert.Contains(t, yarpcerrors.FromError(err).Message(), "CompareAndSwap", "expected Thrift procedure name in message")
}
