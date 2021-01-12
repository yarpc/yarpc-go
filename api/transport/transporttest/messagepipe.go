// Copyright (c) 2021 Uber Technologies, Inc.
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

package transporttest

import (
	"context"
	"io"

	"go.uber.org/yarpc/api/transport"
)

type pipeOptions struct{}

// MessagePipeOption is an option for the MessagePipe constructor.
type MessagePipeOption interface {
	apply(*pipeOptions)
}

type messageResult struct {
	msg *transport.StreamMessage
	err error
}

// MessagePipe creates an in-memory client and server message stream pair.
//
// The third return value is a function that the server side uses to transport
// the end of stream error, if not nil.
// Calling the finish function with nil is valid.
//
//   finish(streamHandler.HandleStream(serverStream))
//
func MessagePipe(ctx context.Context, req *transport.StreamRequest, _ ...MessagePipeOption) (*transport.ClientStream, *transport.ServerStream, func(error), error) {
	c2s := make(chan messageResult)
	s2c := make(chan messageResult)
	clientClosed := make(chan struct{})
	serverClosed := make(chan struct{})
	client, err := transport.NewClientStream(&stream{
		ctx:        ctx,
		req:        req,
		send:       c2s,
		recv:       s2c,
		sendClosed: clientClosed,
		recvClosed: serverClosed,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	server, err := transport.NewServerStream(&stream{
		ctx:        ctx,
		req:        req,
		send:       s2c,
		recv:       c2s,
		sendClosed: serverClosed,
		recvClosed: clientClosed,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	finish := func(err error) {
		if err == nil {
			return
		}
		// If HandleStream returns an error, we realize this
		// by sending that error through the server to client
		// channel, so it can be picked up by the client's next
		// ReceiveMessage/Recv call.
		select {
		case <-clientClosed:
		case <-serverClosed:
		case <-ctx.Done():
		case s2c <- messageResult{err: err}:
			close(serverClosed)
		}
	}

	return client, server, finish, nil
}

type stream struct {
	req        *transport.StreamRequest
	ctx        context.Context
	send       chan<- messageResult
	recv       <-chan messageResult
	sendClosed chan struct{}
	recvClosed chan struct{}
}

func (s *stream) Context() context.Context {
	return s.ctx
}

func (s *stream) Request() *transport.StreamRequest {
	return s.req
}

func (s *stream) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	select {
	case <-s.sendClosed:
		return io.EOF
	case <-s.recvClosed:
		return io.EOF
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	case s.send <- messageResult{msg: msg}:
		return nil
	}
}

func (s *stream) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	select {
	case <-s.sendClosed:
		return nil, io.EOF
	case <-s.recvClosed:
		return nil, io.EOF
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-s.recv:
		return res.msg, res.err
	}
}

func (s *stream) Close(_ context.Context) error {
	close(s.sendClosed)
	return nil
}
