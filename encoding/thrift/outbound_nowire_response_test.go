// Copyright (c) 2026 Uber Technologies, Inc.
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

package thrift

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/protocol/binary"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/clientconfig"
)

func TestNoWireClientResponseReads(t *testing.T) {
	values := make([]string, 1280)
	for i := range values {
		values[i] = strings.Repeat("x", 96)
	}
	payload := encodeStringList(t, values)
	readError := errors.New("response read failed")
	tests := []struct {
		name       string
		reader     io.Reader
		wantError  error
		batchReads bool
	}{
		{name: "stream", reader: bytes.NewReader(payload), batchReads: true},
		{name: "short reads", reader: iotest.OneByteReader(bytes.NewReader(payload))},
		{name: "data with EOF", reader: iotest.DataErrReader(bytes.NewReader(payload))},
		{name: "truncated", reader: bytes.NewReader(payload[:len(payload)-1]), wantError: io.ErrUnexpectedEOF},
		{name: "read failure", reader: io.MultiReader(bytes.NewReader(payload[:128]), iotest.ErrReader(readError)), wantError: readError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			out := transporttest.NewMockUnaryOutbound(ctrl)
			body := &trackedResponseBody{Reader: tt.reader}
			out.EXPECT().Call(gomock.Any(), gomock.Any()).Return(&transport.Response{Body: body}, nil)
			client := NewNoWire(Config{
				Service: "MyService",
				ClientConfig: clientconfig.MultiOutbound("caller", "service", transport.Outbounds{
					Unary: out,
				}),
			})
			var result stringListResponse
			err := client.Call(context.Background(), fakeEnveloper(wire.Call), &result)
			assert.Equal(t, 1, body.closes, "the transport body must close on success and decode failure")
			if tt.wantError != nil {
				require.ErrorIs(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, values, []string(result))
			if tt.batchReads {
				// An unbuffered decoder makes at least two reads per string.
				// Leave room for refills without coupling to an exact buffer size.
				assert.Less(t, body.reads, len(values)/10)
			}
		})
	}
}

func TestNoWireClientResponseAfterDecodeFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	out := transporttest.NewMockUnaryOutbound(ctrl)
	client := NewNoWire(Config{
		Service: "MyService",
		ClientConfig: clientconfig.MultiOutbound("caller", "service", transport.Outbounds{
			Unary: out,
		}),
	})
	decodeError := errors.New("decode rejected response")
	for _, value := range []string{"first", "second", "third"} {
		body := &trackedResponseBody{Reader: bytes.NewReader(encodeStringList(t, []string{value}))}
		out.EXPECT().Call(gomock.Any(), gomock.Any()).Return(&transport.Response{Body: body}, nil)
		result := responseDecoderFunc(func(sr stream.Reader) error {
			// Leave unread data behind in the response reader.
			_, err := sr.ReadListBegin()
			if err != nil {
				return err
			}
			return decodeError
		})
		require.ErrorIs(t, client.Call(context.Background(), fakeEnveloper(wire.Call), result), decodeError)
		assert.Equal(t, 1, body.closes)

		body = &trackedResponseBody{Reader: bytes.NewReader(encodeStringList(t, []string{value}))}
		out.EXPECT().Call(gomock.Any(), gomock.Any()).Return(&transport.Response{Body: body}, nil)
		var decoded stringListResponse
		require.NoError(t, client.Call(context.Background(), fakeEnveloper(wire.Call), &decoded))
		assert.Equal(t, []string{value}, []string(decoded))
		assert.Equal(t, 1, body.closes)
	}
}

func TestNoWireClientSeekableResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	out := transporttest.NewMockUnaryOutbound(ctrl)
	body := &seekableResponseBody{Reader: bytes.NewReader(encodeStringList(t, []string{"skipped"}))}
	out.EXPECT().Call(gomock.Any(), gomock.Any()).Return(&transport.Response{Body: body}, nil)
	client := NewNoWire(Config{
		Service:      "MyService",
		ClientConfig: clientconfig.MultiOutbound("caller", "service", transport.Outbounds{Unary: out}),
	})
	result := responseDecoderFunc(func(sr stream.Reader) error { return sr.Skip(wire.TList) })
	require.NoError(t, client.Call(context.Background(), fakeEnveloper(wire.Call), result))
	assert.Positive(t, body.seeks, "seekable responses must retain the protocol's zero-copy skip path")
	assert.Equal(t, 1, body.closes)
}

type seekableResponseBody struct {
	*bytes.Reader
	seeks  int
	closes int
}

func (b *seekableResponseBody) Seek(offset int64, whence int) (int64, error) {
	b.seeks++
	return b.Reader.Seek(offset, whence)
}

func (b *seekableResponseBody) Close() error {
	b.closes++
	return nil
}

type trackedResponseBody struct {
	io.Reader
	reads  int
	closes int
}

func (b *trackedResponseBody) Read(p []byte) (int, error) {
	b.reads++
	return b.Reader.Read(p)
}

func (b *trackedResponseBody) Close() error {
	b.closes++
	return nil
}

type responseDecoderFunc func(stream.Reader) error

func (f responseDecoderFunc) Decode(sr stream.Reader) error {
	return f(sr)
}

type stringListResponse []string

func (r *stringListResponse) Decode(sr stream.Reader) error {
	list, err := sr.ReadListBegin()
	if err != nil {
		return err
	}
	*r = make([]string, list.Length)
	for i := range *r {
		if (*r)[i], err = sr.ReadString(); err != nil {
			return err
		}
	}
	return sr.ReadListEnd()
}

func encodeStringList(t testing.TB, values []string) []byte {
	t.Helper()
	var payload bytes.Buffer
	sw := binary.NewStreamWriter(&payload)
	require.NoError(t, sw.WriteListBegin(stream.ListHeader{Type: wire.TBinary, Length: len(values)}))
	for _, value := range values {
		require.NoError(t, sw.WriteString(value))
	}
	require.NoError(t, sw.WriteListEnd())
	require.NoError(t, sw.Close())
	return payload.Bytes()
}
