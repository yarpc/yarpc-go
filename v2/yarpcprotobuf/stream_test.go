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

package yarpcprotobuf

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcjson"
	"go.uber.org/yarpc/v2/yarpctest"
)

type mockMessage struct{}

var _ proto.Message = (*mockMessage)(nil)

func (m *mockMessage) Reset()         {}
func (m *mockMessage) ProtoMessage()  {}
func (m *mockMessage) String() string { return "mock" }

func TestReadFromStream(t *testing.T) {
	tests := []struct {
		desc     string
		buffer   *yarpc.Buffer
		encoding yarpc.Encoding
	}{
		{
			desc:     "successful read with proto encoding",
			buffer:   &yarpc.Buffer{},
			encoding: Encoding,
		},
		{
			desc:     "successful read with json encoding",
			buffer:   &yarpc.Buffer{},
			encoding: yarpcjson.Encoding,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx := context.Background()
			stream := yarpctest.NewMockStreamCloser(mockCtrl)

			stream.EXPECT().ReceiveMessage(ctx).Return(tt.buffer, nil)
			stream.EXPECT().Request().Return(
				&yarpc.Request{
					Encoding: tt.encoding,
				},
			)

			clientStream, err := yarpc.NewClientStream(stream)
			require.NoError(t, err)
			_, err = readFromStream(ctx, clientStream, new(mockMessage))
			require.NoError(t, err)
		})
	}
}

func TestReadFromStreamRecieveError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	stream := yarpctest.NewMockStreamCloser(mockCtrl)

	ctx := context.Background()
	giveErr := errors.New("error")
	stream.EXPECT().ReceiveMessage(ctx).Return(nil, giveErr)

	_, gotErr := readFromStream(ctx, stream, new(mockMessage))
	require.EqualError(t, gotErr, giveErr.Error())
}

func TestWriteToStream(t *testing.T) {
	t.Run("invalid encoding", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		ctx := context.Background()
		enc := "raw"

		stream := yarpctest.NewMockStreamCloser(mockCtrl)
		stream.EXPECT().Request().Return(
			&yarpc.Request{
				Encoding: yarpc.Encoding(enc),
			},
		)

		clientStream, err := yarpc.NewClientStream(stream)
		require.NoError(t, err)

		err = writeToStream(ctx, clientStream, nil)
		assert.Equal(t, yarpcerror.New(yarpcerror.CodeInternal, fmt.Sprintf("failed to marshal unexpected encoding %q", enc)), err)
	})

	t.Run("successful write", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		ctx := context.Background()

		stream := yarpctest.NewMockStreamCloser(mockCtrl)
		stream.EXPECT().Request().Return(
			&yarpc.Request{
				Encoding: Encoding,
			},
		)
		stream.EXPECT().SendMessage(ctx, gomock.Any()).Return(nil)

		clientStream, err := yarpc.NewClientStream(stream)
		require.NoError(t, err)
		assert.NoError(t, writeToStream(ctx, clientStream, &mockMessage{}))
	})
}
