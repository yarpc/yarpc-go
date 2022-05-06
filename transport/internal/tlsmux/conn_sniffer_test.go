// Copyright (c) 2022 Uber Technologies, Inc.
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

package tlsmux

import (
	"bytes"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConn struct {
	net.Conn
	buf *bytes.Buffer
}

func (m *mockConn) Read(b []byte) (int, error) {
	return m.buf.Read(b)
}

func TestConnSniffer(t *testing.T) {
	t.Run("must_read_directly_when_not_sniffing", func(t *testing.T) {
		data := []byte("test")
		sniffer := &connSniffer{Conn: &mockConn{buf: bytes.NewBuffer(data)}, disableSniffing: true}

		buf := make([]byte, 4)
		n, err := sniffer.Read(buf)
		require.NoError(t, err, "unexpected error")
		assert.Equal(t, 4, n, "unexpected length")
		assert.Equal(t, data, buf, "unexpected data")
		assert.Zero(t, sniffer.buf.Cap(), "unexpected buffer capacity")
	})

	t.Run("must_store_data_when_sniffing", func(t *testing.T) {
		data := []byte("test")
		sniffer := &connSniffer{Conn: &mockConn{buf: bytes.NewBuffer(data)}}
		require.False(t, sniffer.disableSniffing, "unexpected sniffing value")

		buf := make([]byte, 2)
		n, err := sniffer.Read(buf)
		require.NoError(t, err, "unexpected error")
		assert.Equal(t, 2, n, "unexpected length")
		assert.Equal(t, data[:2], buf, "unexpected data")
		assert.Equal(t, data[:2], sniffer.buf.Bytes(), "unexpected buffer capacity")

		n, err = sniffer.Read(buf)
		require.NoError(t, err, "unexpected error")
		assert.Equal(t, 2, n, "unexpected length")
		assert.Equal(t, data[2:], buf, "unexpected data")
		assert.Equal(t, data, sniffer.buf.Bytes(), "unexpected buffer capacity")
	})

	t.Run("must_empty_buffer_after_sniffing", func(t *testing.T) {
		data := []byte("test")
		sniffer := &connSniffer{
			Conn:            &mockConn{buf: bytes.NewBuffer(data[2:])},
			disableSniffing: true,
			buf:             *bytes.NewBuffer(data[:2]),
		}

		sniffer.stopSniffing()
		require.True(t, sniffer.disableSniffing, "unexpected sniffing value")

		buf := make([]byte, 2)
		n, err := sniffer.Read(buf)
		require.NoError(t, err, "unexpected error")
		assert.Equal(t, 2, n, "unexpected length")
		assert.Equal(t, data[:2], buf, "unexpected data")
		assert.Zero(t, sniffer.buf.Cap(), "unexpected buffer not released")

		n, err = sniffer.Read(buf)
		require.NoError(t, err, "unexpected error")
		assert.Equal(t, 2, n, "unexpected length")
		assert.Equal(t, data[2:], buf, "unexpected data")
	})
}
