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

package grpc

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"go.uber.org/yarpc/api/transport"
	"google.golang.org/grpc"
)

type serverStream struct {
	ctx    context.Context
	treq   *transport.Request
	stream grpc.ServerStream
}

func newServerStream(ctx context.Context, treq *transport.Request, stream grpc.ServerStream) *serverStream {
	return &serverStream{
		ctx:    ctx,
		treq:   treq,
		stream: stream,
	}
}

func (ss *serverStream) Context() context.Context {
	return ss.ctx
}

func (ss *serverStream) Request() *transport.Request {
	return ss.treq
}

func (ss *serverStream) SendMsg(m io.Reader) error {
	msg, err := ioutil.ReadAll(m)
	if err != nil {
		return err
	}
	return ss.stream.SendMsg(msg)
}

func (ss *serverStream) RecvMsg() (io.Reader, error) {
	var msg []byte // TODO performance
	if err := ss.stream.RecvMsg(&msg); err != nil {
		return nil, err
	}
	return bytes.NewReader(msg), nil
}

type clientStream struct {
	ctx    context.Context
	treq   *transport.Request
	stream grpc.ClientStream
}

func newClientStream(ctx context.Context, treq *transport.Request, stream grpc.ClientStream) *clientStream {
	return &clientStream{
		ctx:    ctx,
		treq:   treq,
		stream: stream,
	}
}

func (cs *clientStream) Context() context.Context {
	return cs.ctx
}

func (cs *clientStream) Request() *transport.Request {
	return cs.treq
}

func (cs *clientStream) SendMsg(m io.Reader) error {
	msg, err := ioutil.ReadAll(m)
	if err != nil {
		return err
	}
	return cs.stream.SendMsg(msg)
}

func (cs *clientStream) RecvMsg() (io.Reader, error) {
	var msg []byte // TODO performance
	if err := cs.stream.RecvMsg(&msg); err != nil {
		return nil, err
	}
	return bytes.NewReader(msg), nil
}

func (cs *clientStream) Close() error {
	return cs.stream.CloseSend()
}
