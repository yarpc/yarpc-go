// Copyright (c) 2018 Uber Technologies, Inc.
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

package pally

import "time"

var (
	// Use global no-ops to avoid allocating on each call to NewNop*.
	_nopCounter         Counter         = nop{}
	_nopCounterVector   CounterVector   = nopCounterVec{}
	_nopGauge           Gauge           = nop{}
	_nopGaugeVector     GaugeVector     = nopGaugeVec{}
	_nopLatencies       Latencies       = nop{}
	_nopLatenciesVector LatenciesVector = nopLatenciesVec{}
)

// NewNopCounter returns a no-op Counter.
func NewNopCounter() Counter { return _nopCounter }

// NewNopCounterVector returns a no-op CounterVector.
func NewNopCounterVector() CounterVector { return _nopCounterVector }

// NewNopGauge returns a no-op Gauge.
func NewNopGauge() Gauge { return _nopGauge }

// NewNopGaugeVector returns a no-op GaugeVector.
func NewNopGaugeVector() GaugeVector { return _nopGaugeVector }

// NewNopLatencies returns a no-op Latencies.
func NewNopLatencies() Latencies { return _nopLatencies }

// NewNopLatenciesVector returns a no-op LatenciesVector.
func NewNopLatenciesVector() LatenciesVector { return _nopLatenciesVector }

type nop struct{}

func (nop) Inc() int64              { return 0 }
func (nop) Dec() int64              { return 0 }
func (nop) Add(_ int64) int64       { return 0 }
func (nop) Sub(_ int64) int64       { return 0 }
func (nop) Store(_ int64)           {}
func (nop) Load() int64             { return 0 }
func (nop) Observe(_ time.Duration) {}

type nopCounterVec struct{}

func (nopCounterVec) Get(...string) (Counter, error) { return NewNopCounter(), nil }
func (nopCounterVec) MustGet(...string) Counter      { return NewNopCounter() }

type nopGaugeVec struct{}

func (nopGaugeVec) Get(...string) (Gauge, error) { return NewNopGauge(), nil }
func (nopGaugeVec) MustGet(...string) Gauge      { return NewNopGauge() }

type nopLatenciesVec struct{}

func (nopLatenciesVec) Get(...string) (Latencies, error) { return NewNopLatencies(), nil }
func (nopLatenciesVec) MustGet(...string) Latencies      { return NewNopLatencies() }
