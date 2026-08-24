// Copyright (c) 2026 Uber Technologies, Inc.
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

// Package yabidentity provides inbound middleware that tags and strips the
// identity headers yab (Uber's RPC CLI) stamps on outbound TChannel calls.
//
// yab marks every call it makes with an x-yab-client header, along with a
// best-effort x-yab-source identity derived from the scheduler-injected
// UDEPLOY_SERVICE_NAME environment variable (falling back to the
// self-reported --caller value, and flagging any mismatch between the two
// via x-yab-source-mismatch). None of this is cryptographically verified,
// but unlike --caller it cannot be spoofed via yab flags alone.
//
// This middleware records those headers as a metric so they can be used to
// find and prioritize remaining TChannel callers, then strips them from the
// request before it reaches application code so they can't leak downstream
// through header-forwarding middleware.
package yabidentity

import (
	"context"
	"strconv"

	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/zap"
)

// Header names yab stamps on outbound TChannel calls. These must match the
// constants in yab's transport/tchannel.go exactly.
const (
	yabClientHeader         = "x-yab-client"
	yabSourceHeader         = "x-yab-source"
	yabDestHeader           = "x-yab-dest"
	yabEnvHeader            = "x-yab-env"
	yabSourceMismatchHeader = "x-yab-source-mismatch"
)

const _unknown = "__unknown__"

// Middleware is inbound middleware that records yab caller-identity headers
// as a metric and removes them from the request before calling the next
// handler.
type Middleware struct {
	logger *zap.Logger
	calls  *metrics.CounterVector
}

// New constructs a Middleware. scope is the metrics scope the middleware
// registers its counter on; logger is used to report metric-registration
// failures.
func New(scope *metrics.Scope, logger *zap.Logger) *Middleware {
	if logger == nil {
		logger = zap.NewNop()
	}

	calls, err := scope.CounterVector(metrics.Spec{
		Name:    "yab_calls",
		Help:    "Total number of inbound calls originating from yab, tagged by self-reported identity.",
		VarTags: []string{"source", "env", "source_mismatch"},
	})
	if err != nil {
		logger.Error("Failed to create yab_calls counter vector.", zap.Error(err))
	}

	return &Middleware{logger: logger, calls: calls}
}

// Handle implements middleware.UnaryInbound.
func (m *Middleware) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter, h transport.UnaryHandler) error {
	m.tagAndStrip(req)
	return h.Handle(ctx, req, resw)
}

// HandleOneway implements middleware.OnewayInbound.
func (m *Middleware) HandleOneway(ctx context.Context, req *transport.Request, h transport.OnewayHandler) error {
	m.tagAndStrip(req)
	return h.HandleOneway(ctx, req)
}

// tagAndStrip records the yab identity headers on req, if present, as a
// metric, and then deletes them from req.Headers so they never reach
// application code.
func (m *Middleware) tagAndStrip(req *transport.Request) {
	defer func() {
		req.Headers.Del(yabClientHeader)
		req.Headers.Del(yabSourceHeader)
		req.Headers.Del(yabDestHeader)
		req.Headers.Del(yabEnvHeader)
		req.Headers.Del(yabSourceMismatchHeader)
	}()

	if _, ok := req.Headers.Get(yabClientHeader); !ok {
		return
	}

	source, ok := req.Headers.Get(yabSourceHeader)
	if !ok || source == "" {
		source = _unknown
	}
	env, ok := req.Headers.Get(yabEnvHeader)
	if !ok || env == "" {
		env = _unknown
	}
	_, mismatch := req.Headers.Get(yabSourceMismatchHeader)

	if m.calls == nil {
		return
	}
	counter, err := m.calls.Get(
		"source", source,
		"env", env,
		"source_mismatch", strconv.FormatBool(mismatch),
	)
	if err != nil {
		m.logger.Error("Failed to get yab_calls counter.", zap.Error(err))
		return
	}
	counter.Inc()
}
