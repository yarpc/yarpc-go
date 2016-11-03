// Copyright (c) 2016 Uber Technologies, Inc.
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

package http

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartAddrInUse(t *testing.T) {
	i1 := NewInbound(":0")

	require.NoError(t, i1.Start(transport.ServiceDetail{Name: "foo", Registry: new(transporttest.MockRegistry)}, transport.NoDeps))
	i2 := NewInbound(i1.Addr().String())
	err := i2.Start(transport.ServiceDetail{Name: "foo", Registry: new(transporttest.MockRegistry)}, transport.NoDeps)

	require.Error(t, err)
	oe, ok := err.(*net.OpError)
	assert.True(t, ok && oe.Op == "listen", "expected a listen error")
	if ok {
		se, ok := oe.Err.(*os.SyscallError)
		assert.True(t, ok && se.Syscall == "bind" && se.Err == syscall.EADDRINUSE, "expected a EADDRINUSE bind error")
	}

	assert.NoError(t, i1.Stop())
}

func TestNilAddrAfterStop(t *testing.T) {
	i := NewInbound(":0")
	require.NoError(t, i.Start(transport.ServiceDetail{Name: "foo", Registry: new(transporttest.MockRegistry)}, transport.NoDeps))

	assert.NotEqual(t, ":0", i.Addr().String())
	assert.NotNil(t, i.Addr())
	assert.NoError(t, i.Stop())
	assert.Nil(t, i.Addr())
}

func TestInboundStartAndStop(t *testing.T) {
	i := NewInbound(":0")
	require.NoError(t, i.Start(transport.ServiceDetail{Name: "foo", Registry: new(transporttest.MockRegistry)}, transport.NoDeps))

	assert.NotEqual(t, ":0", i.Addr().String())
	assert.NoError(t, i.Stop())
}

func TestInboundStartError(t *testing.T) {
	err := NewInbound("invalid").Start(transport.ServiceDetail{Name: "foo", Registry: new(transporttest.MockRegistry)}, transport.NoDeps)
	assert.Error(t, err, "expected failure")
}

func TestInboundStopWithoutStarting(t *testing.T) {
	i := NewInbound(":8000")
	assert.Nil(t, i.Addr())
	assert.NoError(t, i.Stop())
}

func TestInboundMux(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("healthy"))
	})

	i := NewInbound(":0", Mux("/rpc/v1", mux))
	h := transporttest.NewMockUnaryHandler(mockCtrl)
	reg := transporttest.NewMockRegistry(mockCtrl)
	require.NoError(t, i.Start(transport.ServiceDetail{Name: "foo", Registry: reg}, transport.NoDeps))

	defer i.Stop()

	addr := fmt.Sprintf("http://%v/", i.Addr().String())
	resp, err := http.Get(addr + "health")
	if assert.NoError(t, err, "/health failed") {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if assert.NoError(t, err, "/health body read error") {
			assert.Equal(t, "healthy", string(body), "/health body mismatch")
		}
	}

	// this should fail
	o := NewOutbound(addr)
	require.NoError(t, o.Start(transport.NoDeps), "failed to start outbound")
	defer o.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = o.CallUnary(ctx, &transport.Request{
		Caller:    "foo",
		Service:   "bar",
		Procedure: "hello",
		Encoding:  raw.Encoding,
		Body:      bytes.NewReader([]byte("derp")),
	})

	if assert.Error(t, err, "RPC call to / should have failed") {
		assert.Equal(t, err.Error(), "404 page not found")
	}

	o = NewOutbound(addr + "rpc/v1")
	require.NoError(t, o.Start(transport.NoDeps), "failed to start outbound")
	defer o.Stop()

	reg.EXPECT().GetHandler("bar", "hello").Return(h, nil)
	h.EXPECT().HandleUnary(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	res, err := o.CallUnary(ctx, &transport.Request{
		Caller:    "foo",
		Service:   "bar",
		Procedure: "hello",
		Encoding:  raw.Encoding,
		Body:      bytes.NewReader([]byte("derp")),
	})

	if assert.NoError(t, err, "expected rpc request to succeed") {
		defer res.Body.Close()
		s, err := ioutil.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Empty(t, s)
		}
	}
}
