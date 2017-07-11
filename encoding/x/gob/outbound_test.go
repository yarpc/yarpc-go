// Copyright (c) 2017 Uber Technologies, Inc.
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

package gob

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"reflect"
	"testing"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/clientconfig"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _typeOfMapInterface = reflect.TypeOf(map[string]interface{}{})

func TestCall(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	caller := "caller"
	service := "service"

	tests := []struct {
		procedure string
		headers   map[string]string
		body      interface{}

		// If nil, want will be used
		resBody []byte

		// whether the outbound receives the request
		noCall bool

		// Either want, or wantType and wantErr must be set.
		want        interface{} // expected response body
		wantHeaders map[string]string
		wantType    reflect.Type // type of response body
		wantErr     string       // error message
	}{
		{
			procedure: "foo",
			body:      []string{"foo", "bar"},
			want:      map[string]interface{}{"success": true},
		},
		{
			procedure: "bar",
			body:      []int{1, 2, 3},
			resBody:   []byte{1, 2, 3},
			wantType:  _typeOfMapInterface,
			wantErr:   `failed to decode "gob" response body for procedure "bar" of service "service"`,
		},
		{
			procedure: "baz",
			body:      func() {}, // funcs cannot be gob.Marshal'ed
			noCall:    true,
			wantType:  _typeOfMapInterface,
			wantErr:   `failed to encode "gob" request body for procedure "baz" of service "service"`,
		},
		{
			procedure:   "requestHeaders",
			headers:     map[string]string{"user-id": "42"},
			body:        map[string]interface{}{},
			want:        map[string]interface{}{},
			wantHeaders: map[string]string{"success": "true"},
		},
	}

	for _, tt := range tests {
		outbound := transporttest.NewMockUnaryOutbound(mockCtrl)
		client := New(clientconfig.MultiOutbound(caller, service,
			transport.Outbounds{
				Unary: outbound,
			}))

		if !tt.noCall {
			var resBody io.Reader
			if tt.resBody != nil {
				resBody = bytes.NewReader(tt.resBody)
			} else {
				resBody = gobFrom(t, tt.want)
			}

			outbound.EXPECT().Call(gomock.Any(),
				transporttest.NewRequestMatcher(t,
					&transport.Request{
						Caller:    caller,
						Service:   service,
						Procedure: tt.procedure,
						Encoding:  Encoding,
						Headers:   transport.HeadersFromMap(tt.headers),
						Body:      gobFrom(t, tt.body),
					}),
			).Return(
				&transport.Response{
					Body:    ioutil.NopCloser(resBody),
					Headers: transport.HeadersFromMap(tt.wantHeaders),
				}, nil)
		}

		var wantType reflect.Type
		if tt.want != nil {
			wantType = reflect.TypeOf(tt.want)
		} else {
			require.NotNil(t, tt.wantType, "wantType is required if want is nil")
			wantType = tt.wantType
		}
		resBody := reflect.New(wantType)

		var (
			opts       []yarpc.CallOption
			resHeaders map[string]string
		)

		for k, v := range tt.headers {
			opts = append(opts, yarpc.WithHeader(k, v))
		}
		opts = append(opts, yarpc.ResponseHeaders(&resHeaders))

		err := client.Call(ctx, tt.procedure, tt.body, resBody.Interface(), opts...)
		if tt.wantErr != "" {
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		} else {
			if assert.NoError(t, err) {
				assert.Equal(t, tt.wantHeaders, resHeaders)
				assert.Equal(t, tt.want, resBody.Elem().Interface())
			}
		}
	}
}

type successAck struct{}

func (a successAck) String() string {
	return "success"
}

func TestCallOneway(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	caller := "caller"
	service := "service"

	tests := []struct {
		procedure string
		headers   map[string]string
		body      interface{}

		// whether the outbound receives the request
		noCall bool

		wantErr string // error message
	}{
		{
			procedure: "foo",
			body:      []string{"foo", "bar"},
		},
		{
			procedure: "baz",
			body:      func() {}, // funcs cannot be gob.Marshal'ed
			noCall:    true,
			wantErr:   `failed to encode "gob" request body for procedure "baz" of service "service"`,
		},
		{
			procedure: "requestHeaders",
			headers:   map[string]string{"user-id": "42"},
			body:      map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		outbound := transporttest.NewMockOnewayOutbound(mockCtrl)
		client := New(clientconfig.MultiOutbound(caller, service,
			transport.Outbounds{
				Oneway: outbound,
			}))

		if !tt.noCall {
			reqMatcher := transporttest.NewRequestMatcher(t,
				&transport.Request{
					Caller:    caller,
					Service:   service,
					Procedure: tt.procedure,
					Encoding:  Encoding,
					Headers:   transport.HeadersFromMap(tt.headers),
					Body:      gobFrom(t, tt.body),
				})

			if tt.wantErr != "" {
				outbound.
					EXPECT().
					CallOneway(gomock.Any(), reqMatcher).
					Return(nil, errors.New(tt.wantErr))
			} else {
				outbound.
					EXPECT().
					CallOneway(gomock.Any(), reqMatcher).
					Return(&successAck{}, nil)
			}
		}

		var opts []yarpc.CallOption

		for k, v := range tt.headers {
			opts = append(opts, yarpc.WithHeader(k, v))
		}

		ack, err := client.CallOneway(ctx, tt.procedure, tt.body, opts...)
		if tt.wantErr != "" {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		} else {
			assert.NoError(t, err, "")
			assert.Equal(t, ack.String(), "success")
		}
	}
}
