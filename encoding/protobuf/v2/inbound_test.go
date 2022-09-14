package v2_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb/v2"
	"go.uber.org/yarpc/encoding/protobuf/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestInboundAnyResolver(t *testing.T) {
	newReq := func() proto.Message { return &testpb.TestMessage{} }
	customAnyResolver := &anyResolver{NewMessage: &testpb.TestMessage{}}
	tests := []struct {
		name     string
		anyURL   string
		resolver v2.AnyResolver
	}{
		{
			name:   "nothing custom",
			anyURL: "uber.yarpc.encoding.protobuf.TestMessage",
		},
		{
			name:     "custom resolver",
			anyURL:   "uber.yarpc.encoding.protobuf.TestMessage",
			resolver: customAnyResolver,
		},
		{
			name:     "custom resolver, custom URL",
			anyURL:   "foo.bar.baz",
			resolver: customAnyResolver,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := v2.NewUnaryHandler(v2.UnaryHandlerParams{
				Handle: func(context.Context, proto.Message) (proto.Message, error) {
					testMessage := &testpb.TestMessage{Value: "foo-bar-baz"}
					any, err := anypb.New(testMessage)
					assert.NoError(t, err)
					any.TypeUrl = tt.anyURL // update to custom URL
					return any, nil
				},
				NewRequest:  newReq,
				AnyResolver: tt.resolver,
			})

			req := &transport.Request{
				Encoding: v2.Encoding,
				Body:     bytes.NewReader(nil),
			}

			var resw transporttest.FakeResponseWriter
			err := handler.Handle(context.Background(), req, &resw)
			require.NoError(t, err)
		})
	}
}
