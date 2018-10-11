// Copyright (c) 2018 Uber Technologies, Inc.
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
package yarpcgrpc

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpctest"
)

// return a streaming handler that echos what it receives
func newTestEchoStreamHandler(name string, numTimes int) []yarpc.TransportProcedure {
	var handler = yarpc.StreamTransportHandlerFunc(func(ss *yarpc.ServerStream) error {
		for i := 0; i < numTimes; i++ {
			msg, err := ss.ReceiveMessage(context.Background())
			if err != nil {
				return err
			}

			err = ss.SendMessage(context.Background(), &yarpc.StreamMessage{
				Body: msg.Body,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return []yarpc.TransportProcedure{
		{
			Name:        name,
			Service:     "test-service",
			HandlerSpec: yarpc.NewStreamTransportHandlerSpec(handler),
		},
	}
}

func TestStream(t *testing.T) {
	const numSends = 10

	// start inbound
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer listener.Close()

	procedures := newTestEchoStreamHandler("test-procedure", numSends)
	inbound := &Inbound{
		Listener: listener,
		Router:   yarpctest.NewFakeRouter(procedures),
	}

	require.NoError(t, inbound.Start(context.Background()))
	defer inbound.Stop(context.Background())

	// start outbound
	dialer := &Dialer{}
	require.NoError(t, dialer.Start(context.Background()))
	defer dialer.Stop(context.Background())

	outbound := &Outbound{
		Dialer: dialer,
		URL:    &url.URL{Host: listener.Addr().String()},
	}

	// init stream
	req := &yarpc.Request{
		Caller:    "test-caller",
		Service:   "test-service",
		Encoding:  "test-encoding",
		Procedure: "test-procedure",
	}

	clientStream, err := outbound.CallStream(context.Background(), req)
	require.NoError(t, err, "could not get client stream")

	// send and receive data
	for i := 0; i < numSends; i++ {
		sendMsg := fmt.Sprintf("hello world! %d", i)
		sendStreamMsg := &yarpc.StreamMessage{
			Body: ioutil.NopCloser(yarpc.NewBufferString(sendMsg)),
		}

		err := clientStream.SendMessage(context.Background(), sendStreamMsg)
		require.NoError(t, err, "could not send message")

		recvMsg, err := clientStream.ReceiveMessage(context.Background())
		require.NoError(t, err)

		recvBytes, err := ioutil.ReadAll(recvMsg.Body)
		require.NoError(t, err)
		assert.Equal(t, sendMsg, string(recvBytes))
	}

	assert.NoError(t, clientStream.Close(context.Background()))
}
