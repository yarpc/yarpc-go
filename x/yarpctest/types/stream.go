// Copyright (c) 2017 Uber Technologies, Inc.
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

package types

import (
	"context"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/x/yarpctest/api"
)

// SendStreamMsg is an action to send a message to a stream.  It can be
// applied to either a server or client stream.
type SendStreamMsg struct {
	api.SafeTestingTBOnStart
	api.NoopStop

	Body        io.ReadCloser
	WantErrMsgs []string
}

// ApplyClientStream implements ClientStreamAction
func (s *SendStreamMsg) ApplyClientStream(t testing.TB, c *transport.ClientStream) {
	s.applyStream(t, c)
}

// ApplyServerStream implements ServerStreamAction
func (s *SendStreamMsg) ApplyServerStream(c *transport.ServerStream) error {
	s.applyStream(s.GetTestingTB(), c)
	return nil
}

func (s *SendStreamMsg) applyStream(t testing.TB, c transport.Stream) {
	err := c.SendMessage(
		context.Background(),
		&transport.StreamMessage{
			Body: s.Body,
		},
	)
	if len(s.WantErrMsgs) > 0 {
		require.Error(t, err)
		for _, wantErrMsg := range s.WantErrMsgs {
			require.Contains(t, err.Error(), wantErrMsg)
		}
		return
	}
	require.NoError(t, err)
	return
}

// RecvStreamMsg is an action to receive a message from a stream.  It can
// be applied to either a server or client stream.
type RecvStreamMsg struct {
	api.SafeTestingTBOnStart
	api.NoopStop

	WantBody          []byte
	WantDecodeErrMsgs []string
	WantErrMsgs       []string
}

// ApplyClientStream implements ClientStreamAction
func (s *RecvStreamMsg) ApplyClientStream(t testing.TB, c *transport.ClientStream) {
	s.applyStream(t, c)
}

// ApplyServerStream implements ServerStreamAction
func (s *RecvStreamMsg) ApplyServerStream(c *transport.ServerStream) error {
	s.applyStream(s.GetTestingTB(), c)
	return nil
}

func (s *RecvStreamMsg) applyStream(t testing.TB, c transport.Stream) {
	msg, err := c.ReceiveMessage(context.Background())
	if len(s.WantErrMsgs) > 0 {
		require.Error(t, err)
		for _, wantErrMsg := range s.WantErrMsgs {
			require.Contains(t, err.Error(), wantErrMsg)
		}
		return
	}
	require.NoError(t, err)

	actualMsg, err := ioutil.ReadAll(msg.Body)
	if len(s.WantDecodeErrMsgs) > 0 {
		require.Error(t, err)
		for _, wantErrMsg := range s.WantDecodeErrMsgs {
			require.Contains(t, err.Error(), wantErrMsg)
		}
		return
	}
	require.NoError(t, err)
	require.Equal(t, s.WantBody, actualMsg, "mismatch on stream messages")
	return
}
