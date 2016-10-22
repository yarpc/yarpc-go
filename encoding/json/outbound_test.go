// Copyright (c) 2016 Uber Technologies, Inc.
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

package json

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"testing"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

var _typeOfMapInterface = reflect.TypeOf(map[string]interface{}{})

func TestCall(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	caller := "caller"
	service := "service"

	tests := []struct {
		procedure       string
		headers         yarpc.Headers
		body            interface{}
		encodedRequest  string
		encodedResponse string

		// whether the outbound receives the request
		noCall bool

		// Either want, or wantType and wantErr must be set.
		want        interface{} // expected response body
		wantHeaders yarpc.Headers
		wantType    reflect.Type // type of response body
		wantErr     string       // error message
	}{
		{
			procedure:       "foo",
			body:            []string{"foo", "bar"},
			encodedRequest:  `["foo","bar"]`,
			encodedResponse: `{"success": true}`,
			want:            map[string]interface{}{"success": true},
		},
		{
			procedure:       "bar",
			body:            []int{1, 2, 3},
			encodedRequest:  `[1,2,3]`,
			encodedResponse: `invalid JSON`,
			wantType:        _typeOfMapInterface,
			wantErr:         `failed to decode "json" response body for procedure "bar" of service "service"`,
		},
		{
			procedure: "baz",
			body:      func() {}, // funcs cannot be json.Marshal'ed
			noCall:    true,
			wantType:  _typeOfMapInterface,
			wantErr:   `failed to encode "json" request body for procedure "baz" of service "service"`,
		},
		{
			procedure:       "requestHeaders",
			headers:         yarpc.NewHeaders().With("user-id", "42"),
			body:            map[string]interface{}{},
			encodedRequest:  "{}",
			encodedResponse: "{}",
			want:            map[string]interface{}{},
			wantHeaders:     yarpc.NewHeaders().With("success", "true"),
		},
	}

	for _, tt := range tests {
		sender := transporttest.NewMockRequestSender(mockCtrl)
		outbound := transporttest.OutboundWithSender(transport.Options{}, sender)
		client := New(transport.IdentityChannel(caller, service, outbound))

		if !tt.noCall {
			sender.EXPECT().Send(gomock.Any(),
				transporttest.NewRequestMatcher(t, &transport.Request{
					Caller:    caller,
					Service:   service,
					Procedure: tt.procedure,
					Encoding:  Encoding,
					Headers:   transport.Headers(tt.headers),
					Body:      bytes.NewReader([]byte(tt.encodedRequest)),
				}),
			).Return(
				&transport.Response{
					Body: ioutil.NopCloser(
						bytes.NewReader([]byte(tt.encodedResponse))),
					Headers: transport.Headers(tt.wantHeaders),
				}, nil)
		}

		var wantType reflect.Type
		if tt.want != nil {
			wantType = reflect.TypeOf(tt.want)
		} else {
			require.NotNil(t, tt.wantType, "wantType is required if want is nil")
			wantType = tt.wantType
		}
		resBody := reflect.Zero(wantType).Interface()

		res, err := client.Call(
			ctx,
			yarpc.NewReqMeta().Procedure(tt.procedure).Headers(tt.headers),
			tt.body,
			&resBody,
		)

		if tt.wantErr != "" {
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		} else {
			if assert.NoError(t, err) {
				assert.Equal(t, tt.wantHeaders, res.Headers())
				assert.Equal(t, tt.want, resBody)
			}
		}
	}
}
