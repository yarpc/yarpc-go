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

// Package http2 provides functionality useful for HTTP/2 transport.
// This source file contains implementations of a server shutdown helper.
package http2

import (
	"context"
	"net"
)

// ShutdownHelper is an interface that provides a shutdown callback intended
// to be used in [Server.RegisterOnShutdown].
type ShutdownHelper interface {
	// TrackConnection is used to track connections.
	// It returns the function to stop tracking conn once closed, or nil if
	// conn is not tracked.
	TrackConnection(conn net.Conn) (untrack func())

	// BeginShutdown starts the shutdown process.
	// Note: It MUST be invoked before [net/http.Server.Shutdown] and so before IsShutdownComplete.
	BeginShutdown()

	// ShutdownCallback is invoked during the server's shutdown process.
	ShutdownCallback()

	// IsShutdownReady indicates whether the first phase of the shutdown
	// process is over on every connection, so that the server can shut down:
	// the first GOAWAY was sent, and its PING acknowledged or timed out.
	// Note: It MUST be invoked after [ShutdownHelper.BeginShutdown], and
	// before [net/http.Server.Shutdown].
	// Warning: This MAY be a blocking operation.
	IsShutdownReady(ctx context.Context) bool

	// IsShutdownComplete indicates whether the shutdown process has completed.
	// Note: It MUST be invoked after [ShutdownHelper.BeginShutdown] has been called.
	// Warning: This MAY be a blocking operation.
	IsShutdownComplete(ctx context.Context) bool
}

// NewShutdownHelper creates an HTTP/2 server shutdown helper.
func NewShutdownHelper() ShutdownHelper {
	return &noopHelper{}
}

var _ ShutdownHelper = (*noopHelper)(nil)

// noopHelper is a no-op implementation of [ShutdownHelper].
type noopHelper struct{}

// TrackConnection implements the [ShutdownHelper] interface for [noopHelper].
func (h *noopHelper) TrackConnection(conn net.Conn) func() { return nil }

// BeginShutdown implements the [ShutdownHelper] interface for [noopHelper].
func (h *noopHelper) BeginShutdown() {}

// ShutdownCallback implements the [ShutdownHelper] interface for [noopHelper].
func (h *noopHelper) ShutdownCallback() {}

// IsShutdownReady implements the [ShutdownHelper] interface for [noopHelper].
func (h *noopHelper) IsShutdownReady(ctx context.Context) bool {
	return true
}

// IsShutdownComplete implements the [ShutdownHelper] interface for [noopHelper].
func (h *noopHelper) IsShutdownComplete(ctx context.Context) bool {
	return true
}
