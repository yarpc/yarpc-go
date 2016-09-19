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

package raw

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/uber/tchannel-go/testutils/testreader"
	"golang.org/x/net/context"
)

func TestCall(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	caller := "caller"
	service := "service"

	tests := []struct {
		procedure    string
		headers      yarpc.Headers
		body         []byte
		responseBody [][]byte

		want        []byte
		wantErr     string
		wantHeaders yarpc.Headers
	}{
		{
			procedure:    "foo",
			body:         []byte{1, 2, 3},
			responseBody: [][]byte{{4}, {5}, {6}},
			want:         []byte{4, 5, 6},
		},
		{
			procedure:    "bar",
			body:         []byte{1, 2, 3},
			responseBody: [][]byte{{4}, {5}, nil, {6}},
			wantErr:      "error set by user",
		},
		{
			procedure:    "headers",
			headers:      yarpc.NewHeaders().With("x", "y"),
			body:         []byte{},
			responseBody: [][]byte{},
			want:         []byte{},
			wantHeaders:  yarpc.NewHeaders().With("a", "b"),
		},
	}

	for _, tt := range tests {
		outbound := transporttest.NewMockOutbound(mockCtrl)
		client := New(transport.IdentityChannel(caller, service, outbound))

		writer, responseBody := testreader.ChunkReader()
		for _, chunk := range tt.responseBody {
			writer <- chunk
		}
		close(writer)

		outbound.EXPECT().Call(gomock.Any(),
			transporttest.NewRequestMatcher(t,
				&transport.Request{
					Caller:    caller,
					Service:   service,
					Procedure: tt.procedure,
					Headers:   transport.Headers(tt.headers),
					Encoding:  Encoding,
					Body:      bytes.NewReader(tt.body),
				}),
		).Return(
			&transport.Response{
				Body:    ioutil.NopCloser(responseBody),
				Headers: transport.Headers(tt.wantHeaders),
			}, nil)

		resBody, res, err := client.Call(
			ctx,
			yarpc.NewReqMeta().Procedure(tt.procedure).Headers(tt.headers),
			tt.body)

		if tt.wantErr != "" {
			if assert.Error(t, err) {
				assert.Equal(t, err.Error(), tt.wantErr)
			}
		} else {
			if assert.NoError(t, err) {
				assert.Equal(t, tt.want, resBody)
				assert.Equal(t, tt.wantHeaders, res.Headers())
			}
		}
	}
}
