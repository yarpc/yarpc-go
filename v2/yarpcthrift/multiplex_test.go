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

package yarpcthrift

import (
	"bytes"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/thriftrw/thrifttest"
	"go.uber.org/thriftrw/wire"
)

func TestMultiplexedEncode(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		service  string
		giveName string
		wantName string
	}{
		{
			service:  "Foo",
			giveName: "bar",
			wantName: "Foo:bar",
		},
		{
			service:  "",
			giveName: "y",
			wantName: ":y",
		},
	}

	for _, tt := range tests {
		mockProto := thrifttest.NewMockProtocol(mockCtrl)
		proto := multiplexedOutboundProtocol{
			Protocol: mockProto,
			Service:  tt.service,
		}

		giveEnvelope := wire.Envelope{
			Value: wire.NewValueStruct(wire.Struct{Fields: []wire.Field{}}),
			Type:  wire.Call,
			Name:  tt.giveName,
			SeqID: 42,
		}

		wantEnvelope := giveEnvelope
		wantEnvelope.Name = tt.wantName
		mockProto.EXPECT().EncodeEnveloped(wantEnvelope, gomock.Any()).Return(nil)

		assert.NoError(t, proto.EncodeEnveloped(giveEnvelope, new(bytes.Buffer)))
	}
}

func TestMultiplexedDecode(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tests := []struct {
		service  string
		giveName string
		wantName string
	}{
		{
			service:  "Foo",
			giveName: "Foo:bar",
			wantName: "bar",
		},
		{
			service:  "Foo",
			giveName: "Bar:baz",
			wantName: "Bar:baz",
		},
	}

	for _, tt := range tests {
		mockProto := thrifttest.NewMockProtocol(mockCtrl)
		proto := multiplexedOutboundProtocol{
			Protocol: mockProto,
			Service:  tt.service,
		}

		mockProto.EXPECT().DecodeEnveloped(gomock.Any()).Return(
			wire.Envelope{
				Value: wire.NewValueStruct(wire.Struct{Fields: []wire.Field{}}),
				Type:  wire.Call,
				Name:  tt.giveName,
				SeqID: 42,
			}, nil)

		gotEnvelope, err := proto.DecodeEnveloped(bytes.NewReader([]byte{}))
		if assert.NoError(t, err) {
			assert.Equal(t, tt.wantName, gotEnvelope.Name)
		}
	}
}
