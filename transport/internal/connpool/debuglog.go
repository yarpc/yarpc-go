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

package connpool

// TEST BRANCH ONLY. Verbose instrumentation for exercising HTTP/2 connection
// scaling by hand. Every message is logged at Info (Debug is not printed) and
// carries the peer, so one peer's pool can be followed through its logs.

import (
	"fmt"
	"sync/atomic"

	"go.uber.org/zap"
)

// String names the state so snapshots are readable.
func (s State) String() string {
	switch s {
	case StateActive:
		return "active"
	case StateDraining:
		return "draining"
	case StateIdle:
		return "idle"
	case StateClosing:
		return "closing"
	default:
		return fmt.Sprintf("state(%d)", int32(s))
	}
}

// testLog logs at Info with the pool's prefix and peer, when it has a logger.
func (p *Pool[T]) testLog(msg string, fields ...zap.Field) {
	if p.Logger == nil {
		return
	}
	all := make([]zap.Field, 0, len(fields)+1)
	all = append(all, zap.String("peer", p.ID))
	all = append(all, fields...)
	p.Logger.Info(p.LogPrefix+": [test] "+msg, all...)
}

// snapshot describes the pool right now: its size, how many connections are
// active, the total in-flight streams, each connection's state and stream
// count, and whether a scale-up is running.
func (p *Pool[T]) snapshot() []zap.Field {
	conns := p.LoadConns()
	desc := make([]string, 0, len(conns))
	active := 0
	var total int32
	for i, c := range conns {
		if c.IsActive() {
			active++
		}
		total += c.StreamCount()
		desc = append(desc, fmt.Sprintf("#%d[%s streams=%d]", i, c.GetState(), c.StreamCount()))
	}
	return []zap.Field{
		zap.Int("conns", len(conns)),
		zap.Int("active_conns", active),
		zap.Int32("total_streams", total),
		zap.Strings("conn_states", desc),
		zap.Bool("scale_up_in_progress", atomic.LoadInt32(&p.isScaling) == 1),
	}
}

// Snapshot logs the pool's snapshot under msg, for callers outside the package.
func (p *Pool[T]) Snapshot(msg string, extra ...zap.Field) {
	if p.Logger == nil {
		return
	}
	p.testLog(msg, append(p.snapshot(), extra...)...)
}
