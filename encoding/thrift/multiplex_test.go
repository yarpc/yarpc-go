package thrift

import (
	"bytes"
	"testing"

	"github.com/thriftrw/thriftrw-go/wire"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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
		mockProto := NewMockProtocol(mockCtrl)
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
		mockProto := NewMockProtocol(mockCtrl)
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
