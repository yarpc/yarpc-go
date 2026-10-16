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

package http2

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewShutdownHelper(t *testing.T) {
	assert.IsType(t, &noopHelper{}, NewShutdownHelper())
}

func TestNoopHelper(t *testing.T) {
	h := NewShutdownHelper()

	t.Run("complete before shutdown", func(t *testing.T) {
		assert.True(t, h.IsShutdownComplete(context.Background()))
	})

	t.Run("shutdown", func(t *testing.T) {
		conn := &stubConn{}
		assert.NotPanics(t, func() {
			h.TrackConnection(conn)
			h.TrackConnection(nil)
			h.BeginShutdown()
			h.ShutdownCallback()
			h.BeginShutdown()
		})
		assert.Empty(t, conn.calls, "must not touch tracked connections")
		assert.True(t, h.IsShutdownComplete(context.Background()))
	})

	t.Run("complete with ended context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.True(t, h.IsShutdownComplete(ctx), "must not block nor fail")
	})
}
