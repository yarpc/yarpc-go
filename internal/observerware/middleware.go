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

package observerware

import (
	"context"
	"sync"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/pally"

	"go.uber.org/zap"
)

var (
	_timeNow      = time.Now // for tests
	_digesterPool = sync.Pool{New: func() interface{} {
		return &digester{make([]byte, 0, 128)}
	}}
	_writerPool = sync.Pool{New: func() interface{} {
		return &writer{}
	}}
)

// writer wraps a transport.ResponseWriter so the observing middleware can
// detect application errors.
type writer struct {
	transport.ResponseWriter

	isApplicationError bool
}

func newWriter(rw transport.ResponseWriter) *writer {
	w := _writerPool.Get().(*writer)
	w.isApplicationError = false
	w.ResponseWriter = rw
	return w
}

func (w *writer) SetApplicationError() {
	w.isApplicationError = true
	w.ResponseWriter.SetApplicationError()
}

func (w *writer) free() {
	_writerPool.Put(w)
}

// A digester creates a null-delimited byte slice from a series of strings. It's
// an efficient way to create map keys.
type digester struct {
	bs []byte
}

// For optimal performance, be sure to free each digester.
func newDigester() *digester {
	d := _digesterPool.Get().(*digester)
	d.bs = d.bs[:0]
	return d
}

func (d *digester) add(s string) {
	if len(d.bs) > 0 {
		// separate labels with a null byte
		d.bs = append(d.bs, '\x00')
	}
	d.bs = append(d.bs, s...)
}

func (d *digester) digest() []byte {
	return d.bs
}

func (d *digester) free() {
	_digesterPool.Put(d)
}

// Middleware is logging and metrics middleware for all RPC types.
type Middleware struct {
	reg     *pally.Registry
	logger  *zap.Logger
	extract ContextExtractor

	// Cache metrics and loggers for each caller-callee-encoding-proc-sk-rk-rd
	// edge in the service graph.
	edgesMu sync.RWMutex
	edges   map[string]*edge
}

// New constructs a Middleware.
func New(logger *zap.Logger, reg *pally.Registry, extract ContextExtractor) *Middleware {
	return &Middleware{
		edges:   make(map[string]*edge, _defaultGraphSize),
		reg:     reg,
		logger:  logger,
		extract: extract,
	}
}

// Handle implements middleware.UnaryInbound.
func (m *Middleware) Handle(ctx context.Context, req *transport.Request, w transport.ResponseWriter, h transport.UnaryHandler) error {
	call := m.begin(ctx, transport.Unary, true /* isInbound */, req)
	wrappedWriter := newWriter(w)
	err := h.Handle(ctx, req, wrappedWriter)
	call.End(err, wrappedWriter.isApplicationError)
	wrappedWriter.free()
	return err
}

// Call implements middleware.UnaryOutbound.
func (m *Middleware) Call(ctx context.Context, req *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	call := m.begin(ctx, transport.Unary, false /* isInbound */, req)
	res, err := out.Call(ctx, req)

	isApplicationError := false
	if res != nil {
		isApplicationError = res.ApplicationError
	}
	call.End(err, isApplicationError)
	return res, err
}

// HandleOneway implements middleware.OnewayInbound.
func (m *Middleware) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	call := m.begin(ctx, transport.Oneway, true /* isInbound */, req)
	err := h.HandleOneway(ctx, req)
	call.End(err, false /* isApplicationError */)
	return err
}

// CallOneway implements middleware.OnewayOutbound.
func (m *Middleware) CallOneway(ctx context.Context, req *transport.Request, out transport.OnewayOutbound) (transport.Ack, error) {
	call := m.begin(ctx, transport.Oneway, false /* isInbound */, req)
	ack, err := out.CallOneway(ctx, req)
	call.End(err, false /* isApplicationError */)
	return ack, err
}

func (m *Middleware) begin(ctx context.Context, rpcType transport.Type, isInbound bool, req *transport.Request) call {
	now := _timeNow()

	d := newDigester()
	d.add(req.Caller)
	d.add(req.Service)
	d.add(string(req.Encoding))
	d.add(req.Procedure)
	d.add(req.ShardKey)
	d.add(req.RoutingKey)
	d.add(req.RoutingDelegate)
	e := m.getOrCreateEdge(d.digest(), req)
	d.free()

	return call{
		edge:    e,
		extract: m.extract,
		started: now,
		ctx:     ctx,
		req:     req,
		rpcType: rpcType,
		inbound: isInbound,
	}
}

func (m *Middleware) getOrCreateEdge(key []byte, req *transport.Request) *edge {
	if e := m.getEdge(key); e != nil {
		return e
	}
	return m.createEdge(key, req)
}

func (m *Middleware) getEdge(key []byte) *edge {
	m.edgesMu.RLock()
	e := m.edges[string(key)]
	m.edgesMu.RUnlock()
	return e
}

func (m *Middleware) createEdge(key []byte, req *transport.Request) *edge {
	m.edgesMu.Lock()
	// Since we'll rarely hit this code path, the overhead of defer is acceptable.
	defer m.edgesMu.Unlock()

	if e, ok := m.edges[string(key)]; ok {
		// Someone beat us to the punch.
		return e
	}

	e := newEdge(m.logger, m.reg, req)
	m.edges[string(key)] = e
	return e
}
