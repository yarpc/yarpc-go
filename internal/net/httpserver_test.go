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

package net

import (
	"context"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/yarpctest"
)

func TestStartAndShutdown(t *testing.T) {
	server := NewHTTPServer(&http.Server{Addr: "127.0.0.1:0"})
	require.NoError(t, server.ListenAndServe())

	require.NotNil(t, server.Listener())
	addr := yarpctest.ZeroAddrToHostPort(server.Listener().Addr())

	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	require.NoError(t, server.Shutdown(context.Background()))
	_, err = net.Dial("tcp", addr)
	require.Error(t, err)
}

func TestStartAddrInUse(t *testing.T) {
	s1 := NewHTTPServer(&http.Server{Addr: "127.0.0.1:0"})
	require.NoError(t, s1.ListenAndServe())
	defer s1.Shutdown(context.Background())

	addr := yarpctest.ZeroAddrToHostPort(s1.Listener().Addr())
	s2 := NewHTTPServer(&http.Server{Addr: addr})
	err := s2.ListenAndServe()

	require.Error(t, err)
	oe, ok := err.(*net.OpError)
	assert.True(t, ok && oe.Op == "listen", "expected a listen error")
	if ok {
		se, ok := oe.Err.(*os.SyscallError)
		assert.True(t, ok && se.Syscall == "bind" && se.Err == syscall.EADDRINUSE, "expected a EADDRINUSE bind error")
	}
}
func TestShutdownAndListen(t *testing.T) {
	server := NewHTTPServer(&http.Server{Addr: "127.0.0.1:0"})
	require.NoError(t, server.ListenAndServe())
	require.NoError(t, server.Shutdown(context.Background()))
	require.Error(t, server.ListenAndServe())
}

func TestShutdownWithoutStart(t *testing.T) {
	server := NewHTTPServer(&http.Server{Addr: "127.0.0.1:0"})
	require.NoError(t, server.Shutdown(context.Background()))
}

func TestStartTwice(t *testing.T) {
	server := NewHTTPServer(&http.Server{Addr: "127.0.0.1:0"})
	require.NoError(t, server.ListenAndServe())
	require.Error(t, server.ListenAndServe())
	require.NoError(t, server.Shutdown(context.Background()))
}

func TestShutdownTwice(t *testing.T) {
	server := NewHTTPServer(&http.Server{Addr: "127.0.0.1:0"})
	require.NoError(t, server.ListenAndServe())
	require.NoError(t, server.Shutdown(context.Background()))
	require.NoError(t, server.Shutdown(context.Background()))
}

func TestListenFail(t *testing.T) {
	server := NewHTTPServer(&http.Server{Addr: "invalid"})
	require.Error(t, server.ListenAndServe())
}

func TestShutdownError(t *testing.T) {
	server := NewHTTPServer(&http.Server{Addr: "127.0.0.1:0"})
	require.NoError(t, server.ListenAndServe())
	require.NoError(t, server.Listener().Close())
	time.Sleep(5 * testtime.Millisecond)
	require.Error(t, server.Shutdown(context.Background()))
}
