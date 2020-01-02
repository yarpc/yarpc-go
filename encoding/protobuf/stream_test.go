// Copyright (c) 2020 Uber Technologies, Inc.
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

package protobuf

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestReadFromStreamDecodeError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	wantErr := errors.New("error")

	stream := transporttest.NewMockStreamCloser(mockCtrl)
	stream.EXPECT().ReceiveMessage(ctx).Return(&transport.StreamMessage{
		Body: ioutil.NopCloser(readErr{err: wantErr}),
	}, nil)
	stream.EXPECT().Request().Return(
		&transport.StreamRequest{
			Meta: &transport.RequestMeta{
				Encoding: Encoding,
			},
		},
	)

	clientStream, err := transport.NewClientStream(stream)
	require.NoError(t, err)

	_, err = readFromStream(ctx, clientStream, func() proto.Message { return nil }, newCodec(nil /*AnyResolver*/))

	assert.Equal(t, wantErr, err)
}

type readErr struct {
	err error
}

func (r readErr) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func TestWriteToStreamInvalidEncoding(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()

	stream := transporttest.NewMockStreamCloser(mockCtrl)
	stream.EXPECT().Request().Return(
		&transport.StreamRequest{
			Meta: &transport.RequestMeta{
				Encoding: transport.Encoding("raw"),
			},
		},
	)

	clientStream, err := transport.NewClientStream(stream)
	require.NoError(t, err)

	err = writeToStream(ctx, clientStream, nil, newCodec(nil /*AnyResolver*/))

	assert.Equal(t, yarpcerrors.Newf(yarpcerrors.CodeInternal, "encoding.Expect should have handled encoding \"raw\" but did not"), err)
}
