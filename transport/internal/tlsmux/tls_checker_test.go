package tlsmux

import (
	"bytes"
	"crypto/tls"
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
		assert.Error(t, err, "unexpected error")
		assert.False(t, isTLS, "unexpected tls")
	})

	t.Run("non_tls_buf_shorter_than_header_length", func(t *testing.T) {
		isTLS, err := isTLSClientHelloRecord(bytes.NewBuffer([]byte{1}))
		assert.NoError(t, err, "unexpected error")
		assert.False(t, isTLS, "unexpected tls")
	})

	t.Run("tls", func(t *testing.T) {
		tlsClientHelloHeader := []byte{22, 3, 1, 0, 238}
		isTLS, err := isTLSClientHelloRecord(bytes.NewBuffer(tlsClientHelloHeader))
		assert.NoError(t, err, "unexpected error")
		assert.True(t, isTLS, "unexpected tls")
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
