package grpc

import (
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"
)

func TestStartAddrInUse(t *testing.T) {
	i1 := NewInbound(50099)
	i2 := NewInbound(50099)

	require.NoError(t, i1.Start(new(transporttest.MockHandler), transport.NoDeps))
	err := i2.Start(new(transporttest.MockHandler), transport.NoDeps)

	require.Error(t, err)
	opErr, ok := err.(*net.OpError)
	assert.True(t, ok && opErr.Op == "listen", "expected a listen error")
	if ok {
		sysErr, ok := opErr.Err.(*os.SyscallError)
		assert.True(t, ok && sysErr.Syscall == "bind" && sysErr.Err == syscall.EADDRINUSE, "expected a EADDRINUSE bind error")
	}

	assert.NoError(t, i1.Stop())
}

func TestInboundStartAndStop(t *testing.T) {
	i := NewInbound(0)

	require.NoError(t, i.Start(new(transporttest.MockHandler), transport.NoDeps))

	serviceInfo := i.Server().GetServiceInfo()
	assert.Equal(t, 1, len(serviceInfo["yarpc"].Methods))
	assert.Equal(t, "yarpc", serviceInfo["yarpc"].Methods[0].Name)

	assert.NoError(t, i.Stop())
}

func TestInboundStartError(t *testing.T) {
	err := NewInbound(-100).Start(new(transporttest.MockHandler), transport.NoDeps)
	// Verify that two inbounds started on the same port
	assert.Error(t, err, "expected failure")
}

func TestInboundStopWithoutStarting(t *testing.T) {
	i := NewInbound(8000)

	assert.Nil(t, i.Server())
	assert.NoError(t, i.Stop())
}
