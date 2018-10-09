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
	"io"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcjson"
	"go.uber.org/yarpc/v2/yarpctest"
)

type mockReader struct {
	closeErr error
	readErr  error
}

func (r *mockReader) Read(_ []byte) (n int, err error) {
	if r.readErr != nil {
		return 0, r.readErr
	}
	bytes, err := proto.Marshal(&mockMessage{})
	if err != nil {
		return 0, err
	}
	return len(bytes), io.EOF
}

func (r *mockReader) Close() error {
	return r.closeErr
}

type mockMessage struct{}

var _ proto.Message = (*mockMessage)(nil)

func (m *mockMessage) Reset()         {}
func (m *mockMessage) ProtoMessage()  {}
func (m *mockMessage) String() string { return "mock" }

func TestReadFromStream(t *testing.T) {
	_closeErr := errors.New("faild to close")
	_readErr := errors.New("faild to read")

	tests := []struct {
		desc     string
		reader   *mockReader
		encoding yarpc.Encoding
		err      error
	}{
		{
			desc:     "decode error",
			reader:   &mockReader{readErr: _readErr},
			encoding: Encoding,
			err:      _readErr,
		},
		{
			desc:     "close error",
			reader:   &mockReader{closeErr: _closeErr},
			encoding: Encoding,
			err:      _closeErr,
		},
		{
			desc:     "decode and close multierror",
			reader:   &mockReader{readErr: _readErr, closeErr: _closeErr},
			encoding: Encoding,
			err:      multierr.Append(_readErr, _closeErr),
		},
		{
			desc:     "successful read with proto encoding",
			reader:   &mockReader{},
			encoding: Encoding,
		},
		{
			desc:     "successful read with json encoding",
			reader:   &mockReader{},
			encoding: yarpcjson.Encoding,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			ctx := context.Background()
			stream := yarpctest.NewMockStreamCloser(mockCtrl)

			stream.EXPECT().ReceiveMessage(ctx).Return(
				&yarpc.StreamMessage{
					Body: tt.reader,
				},
				nil,
			)
			stream.EXPECT().Request().Return(
				&yarpc.Request{
					Encoding: tt.encoding,
				},
			)

			clientStream, err := yarpc.NewClientStream(stream)
			require.NoError(t, err)

			_, err = readFromStream(ctx, clientStream, new(mockMessage))
			assert.Equal(t, tt.err, err)
		})
	}
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
		assert.Equal(t, yarpcerror.Newf(yarpcerror.CodeInternal, "failed to marshal unexpected encoding %q", enc), err)
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
