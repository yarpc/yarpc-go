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
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestStartAddrInUse(t *testing.T) {
	i1 := NewInbound(":0")
	require.NoError(t, i1.Start(new(transporttest.MockHandler)))
	i2 := NewInbound(i1.Addr().String())
	err := i2.Start(new(transporttest.MockHandler))

	oe, ok := err.(*net.OpError)
	assert.True(t, ok && oe.Op == "listen", "expected a listen error")
	if ok {
		se, ok := oe.Err.(*os.SyscallError)
		assert.True(t, ok && se.Syscall == "bind" && se.Err == syscall.EADDRINUSE, "expected a EADDRINUSE bind error")
	}

	assert.Error(t, err)
	assert.NoError(t, i1.Stop())
}

func TestInboundStartAndStop(t *testing.T) {
	i := NewInbound(":0")
	require.NoError(t, i.Start(new(transporttest.MockHandler)))
	assert.NotEqual(t, ":0", i.Addr().String())
	assert.NoError(t, i.Stop())
}

func TestInboundStartError(t *testing.T) {
	err := NewInbound("invalid").Start(new(transporttest.MockHandler))
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

	i := NewInbound(":8080", Mux("/rpc/v1", mux))
	h := transporttest.NewMockHandler(mockCtrl)
	require.NoError(t, i.Start(h))
	defer i.Stop()

	addr := "http://127.0.0.1:8080/"
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
	_, err = o.Call(context.TODO(), &transport.Request{
		Caller:    "foo",
		Service:   "bar",
		Procedure: "hello",
		Encoding:  raw.Encoding,
		Body:      bytes.NewReader([]byte("derp")),
		TTL:       time.Second,
	})

	if assert.Error(t, err, "RPC call to / should have failed") {
		assert.Equal(t, err.Error(), "404 page not found")
	}

	o = NewOutbound(addr + "rpc/v1")
	h.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	res, err := o.Call(context.TODO(), &transport.Request{
		Caller:    "foo",
		Service:   "bar",
		Procedure: "hello",
		Encoding:  raw.Encoding,
		Body:      bytes.NewReader([]byte("derp")),
		TTL:       time.Second,
	})

	if assert.NoError(t, err, "expected rpc request to succeed") {
		defer res.Body.Close()
		s, err := ioutil.ReadAll(res.Body)
		if assert.NoError(t, err) {
			assert.Empty(t, s)
		}
	}
}
