// Copyright (c) 2019 Uber Technologies, Inc.
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

package observability

import (
	"context"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
)

const (
	_successfulStreamReceive = "Successfully received stream message"
	_successfulStreamSend    = "Successfully sent stream message"
	_errorStreamReceive      = "Error receiving stream message"
	_errorStreamSend         = "Error sending stream message"
)

var _ transport.StreamCloser = (*streamWrapper)(nil)

type streamWrapper struct {
	transport.StreamCloser

	call   call
	edge   *streamEdge
	logger *zap.Logger
}

func (c call) WrapClientStream(stream *transport.ClientStream) *transport.ClientStream {
	wrapped, err := transport.NewClientStream(&streamWrapper{
		StreamCloser: stream,
		call:         c,
		edge:         c.edge.streaming,
		logger:       c.edge.logger,
	})
	if err != nil {
		// This will never happen since transport.NewClientStream only returns an
		// error for nil streams. In the nearly impossible situation where it does,
		// we fall back to using the original, unwrapped stream.
		c.edge.logger.DPanic("transport.ClientStream wrapping should never fail, streaming metrics are disabled")
		wrapped = stream
	}
	return wrapped
}

func (c call) WrapServerStream(stream *transport.ServerStream) *transport.ServerStream {
	wrapped, err := transport.NewServerStream(&streamWrapper{
		StreamCloser: nopCloser{stream},
		call:         c,
		edge:         c.edge.streaming,
		logger:       c.edge.logger,
	})
	if err != nil {
		// This will never happen since transport.NewServerStream only returns an
		// error for nil streams. In the nearly impossible situation where it does,
		// we fall back to using the original, unwrapped stream.
		c.edge.logger.DPanic("transport.ServerStream wrapping should never fail, streaming metrics are disabled")
		wrapped = stream
	}
	return wrapped
}

func (s *streamWrapper) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	err := s.StreamCloser.SendMessage(ctx, msg)
	s.call.logStreamEvent(err, _successfulStreamSend, _errorStreamSend)

	s.edge.sends.Inc()
	if err == nil {
		s.edge.sendSuccesses.Inc()
		return nil
	}

	if sendFailuresCounter, err2 := s.edge.sendFailures.Get(_error, errToMetricString(err)); err2 != nil {
		s.logger.DPanic("could not retrieve send failure counter", zap.Error(err2))
	} else {
		sendFailuresCounter.Inc()
	}
	return err
}

func (s *streamWrapper) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	msg, err := s.StreamCloser.ReceiveMessage(ctx)
	s.call.logStreamEvent(err, _successfulStreamReceive, _errorStreamReceive)

	s.edge.receives.Inc()
	if err == nil {
		s.edge.receiveSuccesses.Inc()
		return msg, nil
	}

	if recvFailureCounter, err2 := s.edge.receiveFailures.Get(_error, errToMetricString(err)); err2 != nil {
		s.logger.DPanic("could not retrieve receive failure counter", zap.Error(err2))
	} else {
		recvFailureCounter.Inc()
	}

	return msg, err
}

func (s *streamWrapper) Close(ctx context.Context) error {
	err := s.StreamCloser.Close(ctx)
	s.call.EndStream(err)
	return err
}

// This is a light wrapper so that we can re-use the same methods for
// instrumenting observability. The transport.ClientStream has an additional
// Close(ctx) method, unlike the transport.ServerStream.
type nopCloser struct {
	transport.Stream
}

func (c nopCloser) Close(ctx context.Context) error {
	return nil
}
