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
	"crypto/tls"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTLSClientHelloRecord(t *testing.T) {
	t.Run("non_tls", func(t *testing.T) {
		isTLS, err := isTLSClientHelloRecord(bytes.NewBuffer([]byte("testtls")))
		assert.NoError(t, err, "unexpected error")
		assert.False(t, isTLS, "unexpected tls")
	})

	t.Run("read_error", func(t *testing.T) {
		isTLS, err := isTLSClientHelloRecord(bytes.NewBuffer(nil))
		assert.Error(t, err, "unexpected success")
		assert.False(t, isTLS, "unexpected tls")
	})

	t.Run("non_tls_buf_shorter_than_header_length", func(t *testing.T) {
		isTLS, err := isTLSClientHelloRecord(bytes.NewBuffer([]byte{1}))
		assert.NoError(t, err, "unexpected error")
		assert.False(t, isTLS, "unexpected tls")
	})

	t.Run("tls_header", func(t *testing.T) {
		tests := []struct {
			minorVersion  int
			expectSuccess bool
		}{
			{minorVersion: 0, expectSuccess: false},
			{minorVersion: 1, expectSuccess: true},
			{minorVersion: 2, expectSuccess: true},
			{minorVersion: 3, expectSuccess: true},
			{minorVersion: 4, expectSuccess: true},
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("minor_version_%d", tt.minorVersion), func(t *testing.T) {
				tlsClientHelloHeader := []byte{22, 3, byte(tt.minorVersion), 0, 238}
				isTLS, err := isTLSClientHelloRecord(bytes.NewBuffer(tlsClientHelloHeader))
				assert.NoError(t, err, "unexpected error")
				if tt.expectSuccess {
					assert.True(t, isTLS, "unexpected non tls")
				} else {
					assert.False(t, isTLS, "unexpected tls")
				}
			})
		}
	})

	t.Run("e2e_tls", func(t *testing.T) {
		lis, err := net.Listen("tcp", ":0")
		require.NoError(t, err)
		defer lis.Close()

		blockCh := make(chan struct{})
		go func() {
			tls.Dial("tcp", lis.Addr().String(), &tls.Config{})
			close(blockCh)
		}()

		conn, err := lis.Accept()
		require.NoError(t, err, "unexpected accept error")

		isTLS, err := isTLSClientHelloRecord(conn)
		assert.NoError(t, err, "unexpected error")
		assert.True(t, isTLS, "unexpected istls")

		conn.Close()
		<-blockCh
	})
}
