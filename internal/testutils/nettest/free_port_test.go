package nettest

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustGetFreeHostPort(t *testing.T) {
	hostPort := MustGetFreeHostPort()
	ln, err := net.Listen("tcp", hostPort)
	require.NoError(t, err, "Failed to listen on %v", hostPort)
	assert.Equal(t, hostPort, ln.Addr().String(), "Listening on wrong host:port")
	require.NoError(t, ln.Close(), "Failed to close listener")
}

func TestMustGetFreePort(t *testing.T) {
	port := MustGetFreePort()
	listenOn := fmt.Sprintf("127.0.0.1:%v", port)
	ln, err := net.Listen("tcp", listenOn)
	require.NoError(t, err, "Failed to listen on %v", listenOn)
	assert.Equal(t, listenOn, ln.Addr().String(), "Listening on wrong host:port")
	require.NoError(t, ln.Close(), "Failed to close listener")
}
