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
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

var _ transport.StreamCloser = (*streamWrapper)(nil)

type streamWrapper struct {
	call   call
	stream transport.StreamCloser
}

func newClientStreamWrapper(call call, stream transport.StreamCloser) transport.StreamCloser {
	return &streamWrapper{
		call:   call,
		stream: stream,
	}
}

func newServerStreamWrapper(call call, stream transport.Stream) transport.Stream {
	return &streamWrapper{
		call:   call,
		stream: contextCloser{stream},
	}
}

func (s *streamWrapper) Context() context.Context {
	return s.stream.Context()
}

func (s *streamWrapper) Request() *transport.StreamRequest {
	return s.stream.Request()
}

func (s *streamWrapper) SendMessage(ctx context.Context, msg *transport.StreamMessage) error {
	edge := s.call.edge.streaming

	start := _timeNow()
	err := s.stream.SendMessage(ctx, msg)
	elapsed := _timeNow().Sub(start)

	edge.sends.Inc()
	if err == nil {
		edge.sendSuccesses.Inc()
		edge.sendLatencies.Observe(elapsed)

	} else {
		if sendFailuresCounter, err2 := edge.sendFailures.Get(_error, errToMetricString(err)); err2 != nil {
			s.call.edge.logger.DPanic("could not retrieve send failure counter", zap.Error(err2))
		} else {
			sendFailuresCounter.Inc()
		}
		edge.sendErrLatencies.Observe(elapsed)
	}

	s.call.log(elapsed, err, false /* application error bit */)
	return err
}

func (s *streamWrapper) ReceiveMessage(ctx context.Context) (*transport.StreamMessage, error) {
	edge := s.call.edge.streaming

	start := _timeNow()
	msg, err := s.stream.ReceiveMessage(ctx)
	elapsed := _timeNow().Sub(start)

	edge.receives.Inc()
	if err == nil {
		edge.receiveSuccesses.Inc()
		edge.receiveLatencies.Observe(elapsed)

	} else {
		if recvFailureCounter, err2 := edge.receiveFailures.Get(_error, errToMetricString(err)); err2 != nil {
			s.call.edge.logger.DPanic("could not retrieve receive failure counter", zap.Error(err2))
		} else {
			recvFailureCounter.Inc()
		}
		edge.recieveErrLatencies.Observe(elapsed)
	}

	s.call.log(elapsed, err, false /* application error bit */)
	return msg, err
}

func (s *streamWrapper) Close(ctx context.Context) error {
	err := s.stream.Close(ctx)
	s.call.EndStream(err)
	return err
}

// This is a light wrapper so that we can re-use the same methods for
// instrumenting observaiblity. The transport.ServerStream does not have a
// Close(ctx) method, unlike the transport.ClientStream.
type contextCloser struct {
	transport.Stream
}

func (c contextCloser) Close(ctx context.Context) error {
	return nil
}

// inteded for metric tags, this returns the yarpcerrors.Status error code name
// or "unknown_internal_yarpc"
func errToMetricString(err error) string {
	if yarpcerrors.IsStatus(err) {
		return yarpcerrors.FromError(err).Code().String()
	}
	return "unknown_internal_yarpc"
}
