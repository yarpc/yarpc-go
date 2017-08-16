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

package raw

import (
	"bytes"
	"io/ioutil"

	"go.uber.org/yarpc/api/transport"
)

// ServerStream exposes functions to interact with a Stream using raw
// primitives.
type ServerStream struct {
	stream transport.ServerStream
}

func newServerStream(stream transport.ServerStream) *ServerStream {
	return &ServerStream{
		stream: stream,
	}
}

// Receive returns a []byte from the stream. It will block until the stream
// responds.
func (ss *ServerStream) Receive() ([]byte, error) {
	reader, err := ss.stream.RecvMsg()
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(reader)
}

// Send sends a byte array over a stream.  It will block until the bytes are
// sent.
func (ss *ServerStream) Send(m []byte) error {
	return ss.stream.SendMsg(bytes.NewReader(m))
}

// ClientStream exposes functions to interact with a Stream using raw
// primitives.
type ClientStream struct {
	stream transport.ClientStream
}

func newClientStream(stream transport.ClientStream) *ClientStream {
	return &ClientStream{
		stream: stream,
	}
}

// Receive returns a []byte from the stream. It will block until the stream
// responds.
func (cs *ClientStream) Receive() ([]byte, error) {
	reader, err := cs.stream.RecvMsg()
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(reader)
}

// Send sends a byte array over a stream.  It will block until the bytes are
// sent.
func (cs *ClientStream) Send(m []byte) error {
	return cs.stream.SendMsg(bytes.NewReader(m))
}

// Close closes the stream.
func (cs *ClientStream) Close() error {
	return cs.stream.Close()
}
