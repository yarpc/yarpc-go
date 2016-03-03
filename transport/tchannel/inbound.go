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

package tchannel

import (
	"net"
	"time"

	"github.com/yarpc/yarpc-go/transport"

	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
)

// Inbound represents a TChannel Inbound. It is the same as a transport
// Inbound except it exposes the address on which the system is listening for
// connections.
type Inbound interface {
	transport.Inbound

	// Address on which the server is listening. Returns nil if Start has not
	// been called yet.
	Addr() net.Addr
}

// InboundOption configures Inbound.
type InboundOption func(*inbound)

// ListenAddr changes the address on which the TChannel server will listen for
// connections. By default, the server listens on an OS-assigned port.
//
// This option has no effect if the Chanel provided to NewInbound is already
// listening for connections when Start() is called.
func ListenAddr(addr string) InboundOption {
	return func(i *inbound) { i.addr = addr }
}

// NewInbound builds a new TChannel inbound from the given Channel.
func NewInbound(ch *tchannel.Channel, opts ...InboundOption) Inbound {
	i := &inbound{ch: ch}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

type inbound struct {
	ch       *tchannel.Channel
	addr     string
	listener net.Listener
}

func (i *inbound) Start(h transport.Handler) error {
	i.ch.GetSubChannel(i.ch.ServiceName()).SetHandler(handler{h})

	if i.ch.State() == tchannel.ChannelListening {
		// Channel.Start() was called before RPC.Start(). We still want to
		// update the Handler and what i.addr means, but nothing else.
		i.addr = i.listener.Addr().String()
		return nil
	}

	// Default to ListenIP if addr wasn't given.
	addr := i.addr
	if addr == "" {
		listenIP, err := tchannel.ListenIP()
		if err != nil {
			return err
		}

		addr = listenIP.String() + ":0"
		// TODO(abg): Find a way to export this to users
	}

	// TODO(abg): If addr was just the port (":4040"), we want to use
	// ListenIP() + ":4040" rather than just ":4040".

	var err error
	i.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	i.addr = i.listener.Addr().String() // in case it changed

	if err := i.ch.Serve(i.listener); err != nil {
		return err
	}

	return nil
}

func (i *inbound) Stop() error {
	i.ch.Close()
	return nil
}

func (i *inbound) Addr() net.Addr {
	if i.listener == nil {
		return nil
	}
	return i.listener.Addr()
}

type handler struct{ Handler transport.Handler }

func (h handler) Handle(ctx context.Context, call *tchannel.InboundCall) {
	deadline, ok := ctx.Deadline()
	if !ok {
		call.Response().SendSystemError(tchannel.ErrTimeoutRequired)
		return
	}

	headers, err := readHeaders(call.Format(), call.Arg2Reader)
	if err != nil {
		call.Response().SendSystemError(tchannel.NewSystemError(
			tchannel.ErrCodeUnexpected, "failed to read headers: %v", err))
		return
	}

	body, err := call.Arg3Reader()
	if err != nil {
		call.Response().SendSystemError(tchannel.NewSystemError(
			tchannel.ErrCodeUnexpected, "failed to read body: %v", err))
		return
	}
	defer body.Close()

	rw := newResponseWriter(call)
	defer rw.Close()

	treq := &transport.Request{
		Caller:    call.CallerName(),
		Service:   call.ServiceName(),
		Encoding:  transport.Encoding(call.Format()),
		Procedure: call.MethodString(),
		Headers:   headers,
		Body:      body,
		TTL:       deadline.Sub(time.Now()),
	}

	if err := h.Handler.Handle(ctx, treq, rw); err != nil {
		call.Response().SendSystemError(tchannel.NewSystemError(
			tchannel.ErrCodeUnexpected, "internal error: %v", err))
		return
	}
}

type responseWriter struct {
	failedWith   error
	bodyWriter   tchannel.ArgWriter
	format       tchannel.Format
	headers      transport.Headers
	response     *tchannel.InboundCallResponse
	wroteHeaders bool
}

func newResponseWriter(call *tchannel.InboundCall) *responseWriter {
	return &responseWriter{
		response: call.Response(),
		headers:  make(transport.Headers),
		format:   call.Format(),
	}
}

func (rw *responseWriter) AddHeaders(h transport.Headers) {
	if rw.wroteHeaders {
		panic("AddHeaders() cannot be called after calling Write().")
	}
	for k, v := range h {
		rw.headers.Set(k, v)
	}
}

func (rw *responseWriter) Write(s []byte) (int, error) {
	if rw.failedWith != nil {
		return 0, rw.failedWith
	}

	if !rw.wroteHeaders {
		rw.wroteHeaders = true
		if err := writeHeaders(rw.format, rw.headers, rw.response.Arg2Writer); err != nil {
			rw.failedWith = err
			return 0, err
		}
	}

	if rw.bodyWriter == nil {
		var err error
		rw.bodyWriter, err = rw.response.Arg3Writer()
		if err != nil {
			rw.failedWith = err
			return 0, err
		}
	}

	return rw.bodyWriter.Write(s)
}

func (rw *responseWriter) Close() {
	if rw.bodyWriter != nil {
		rw.bodyWriter.Close()
		rw.bodyWriter = nil
	}
}
