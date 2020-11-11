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

package grpc

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"github.com/gogo/status"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/grpcerrorcodes"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	_ transport.StreamHeadersSender = (*serverStream)(nil)
	_ transport.StreamHeadersReader = (*clientStream)(nil)
)

type serverStream struct {
	ctx    context.Context
	req    *transport.StreamRequest
	stream grpc.ServerStream
}

func newServerStream(ctx context.Context, req *transport.StreamRequest, stream grpc.ServerStream) *serverStream {
	return &serverStream{
		ctx:    ctx,
		req:    req,
		stream: stream,
	}
}

func (ss *serverStream) Context() context.Context {
	return ss.ctx
}

func (ss *serverStream) Request() *transport.StreamRequest {
	return ss.req
}

func (ss *serverStream) SendMessage(_ context.Context, m *transport.StreamMessage) error {
	// TODO pool buffers for performance.
	msg, err := ioutil.ReadAll(m.Body)
	_ = m.Body.Close()
	if err != nil {
		return err
	}
	return toYARPCStreamError(ss.stream.SendMsg(msg))
}

func (ss *serverStream) ReceiveMessage(_ context.Context) (*transport.StreamMessage, error) {
	var msg []byte
	if err := ss.stream.RecvMsg(&msg); err != nil {
		return nil, toYARPCStreamError(err)
	}
	return &transport.StreamMessage{
		Body:     readCloser{bytes.NewReader(msg)},
		BodySize: len(msg),
	}, nil
}

type readCloser struct {
	*bytes.Reader
}

func (r readCloser) Close() error {
	return nil
}

func (ss *serverStream) SendHeaders(headers transport.Headers) error {
	md := make(metadata.MD, headers.Len())
	for k, v := range headers.Items() {
		md.Set(k, v)
	}
	return ss.stream.SendHeader(md)
}

type clientStream struct {
	ctx     context.Context
	req     *transport.StreamRequest
	stream  grpc.ClientStream
	span    opentracing.Span
	closed  atomic.Bool
	release func(error)
}

func newClientStream(ctx context.Context, req *transport.StreamRequest, stream grpc.ClientStream, span opentracing.Span, release func(error)) *clientStream {
	return &clientStream{
		ctx:     ctx,
		req:     req,
		stream:  stream,
		span:    span,
		release: release,
	}
}

func (cs *clientStream) Context() context.Context {
	return cs.ctx
}

func (cs *clientStream) Request() *transport.StreamRequest {
	return cs.req
}

func (cs *clientStream) SendMessage(_ context.Context, m *transport.StreamMessage) error {
	if cs.closed.Load() { // If the stream is closed, we should not be sending messages on it.
		return io.EOF
	}
	// TODO can we make a "Bytes" interface to get direct access to the bytes
	// (instead of resorting to ReadAll (which is not necessarily performant))
	msg, err := ioutil.ReadAll(m.Body)
	_ = m.Body.Close()
	if err != nil {
		return toYARPCStreamError(err)
	}
	if err := cs.stream.SendMsg(msg); err != nil {
		return toYARPCStreamError(cs.closeWithErr(err))
	}
	return nil
}

func (cs *clientStream) ReceiveMessage(context.Context) (*transport.StreamMessage, error) {
	// TODO use buffers for performance reasons.
	var msg []byte
	if err := cs.stream.RecvMsg(&msg); err != nil {
		return nil, toYARPCStreamError(cs.closeWithErr(err))
	}
	return &transport.StreamMessage{Body: ioutil.NopCloser(bytes.NewReader(msg))}, nil
}

func (cs *clientStream) Close(context.Context) error {
	_ = cs.closeWithErr(nil)
	return cs.stream.CloseSend()
}

func (cs *clientStream) Headers() (transport.Headers, error) {
	md, err := cs.stream.Header()
	if err != nil {
		return transport.NewHeaders(), err
	}
	headers := transport.NewHeadersWithCapacity(len(md))
	for k, vs := range md {
		if len(vs) > 0 {
			headers = headers.With(k, vs[0])
		}
	}
	return headers, nil
}

func (cs *clientStream) closeWithErr(err error) error {
	if !cs.closed.Swap(true) {
		err = transport.UpdateSpanWithErr(cs.span, err)
		cs.span.Finish()
		cs.release(err)
	}
	return err
}

func toYARPCStreamError(err error) error {
	if err == nil {
		return nil
	}
	if err == io.EOF {
		return err
	}
	if yarpcerrors.IsStatus(err) {
		return err
	}
	status, ok := status.FromError(err)
	// if not a yarpc error or grpc error, just return a wrapped error
	if !ok {
		return yarpcerrors.FromError(err)
	}
	code, ok := grpcerrorcodes.GRPCCodeToYARPCCode[status.Code()]
	if !ok {
		code = yarpcerrors.CodeUnknown
	}
	yarpcerr := yarpcerrors.Newf(code, status.Message())
	details, err := marshalError(status)
	if err != nil {
		return yarpcerrors.FromError(err)
	}
	if details != nil {
		yarpcerr = yarpcerr.WithDetails(details)
	}
	return yarpcerr
}
