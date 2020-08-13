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

package transport_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
)

func TestServerStreamHeaders(t *testing.T) {
	items := map[string]string{"header-key": "header-value"}
	headers := transport.HeadersFromMap(items)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStream := transporttest.NewMockStream(mockCtrl)

	t.Run("unimplemented", func(t *testing.T) {
		serverStream, err := transport.NewServerStream(mockStream)
		require.NoError(t, err)

		err = serverStream.SendHeaders(headers)
		assert.Error(t, err)
	})

	t.Run("send-headers", func(t *testing.T) {
		fakeWriter := &fakeWriter{
			Stream: mockStream,
		}
		serverStream, err := transport.NewServerStream(fakeWriter)
		require.NoError(t, err)

		err = serverStream.SendHeaders(headers)
		assert.NoError(t, err)
		assert.Equal(t, items, fakeWriter.headers.Items())
	})
}

var _ transport.StreamHeadersSender = (*fakeWriter)(nil)

type fakeWriter struct {
	transport.Stream

	headers transport.Headers
}

func (fw *fakeWriter) SendHeaders(headers transport.Headers) error {
	fw.headers = headers
	return nil
}

func TestClientStreamHeaders(t *testing.T) {
	items := map[string]string{"header-key": "header-value"}
	headers := transport.HeadersFromMap(items)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockStream := transporttest.NewMockStreamCloser(mockCtrl)

	t.Run("unimplemented", func(t *testing.T) {
		clientStream, err := transport.NewClientStream(mockStream)
		require.NoError(t, err)

		_, err = clientStream.Headers()
		assert.Error(t, err)
	})

	t.Run("send-headers", func(t *testing.T) {
		fakeReader := &fakeReader{
			StreamCloser: mockStream,
			headers:      headers,
		}
		clientStream, err := transport.NewClientStream(fakeReader)
		require.NoError(t, err)

		headers, err = clientStream.Headers()
		assert.NoError(t, err)
		assert.Equal(t, items, fakeReader.headers.Items())
	})
}

var _ transport.StreamHeadersReader = (*fakeReader)(nil)

type fakeReader struct {
	transport.StreamCloser

	headers transport.Headers
}

func (fr *fakeReader) Headers() (transport.Headers, error) {
	return fr.headers, nil
}
