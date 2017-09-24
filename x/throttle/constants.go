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

package throttle

import "time"

const (
	// Sleep between pushes to Tally metrics. At some point, we may want this
	// to be configurable.
	_tallyPushInterval = 500 * time.Millisecond
	_packageName       = "yarpc-throttle"
)

var (
	// Latency buckets for the overhead latency histogram.
	_ms      = time.Millisecond
	_buckets = []time.Duration{
		1 * _ms,
		2 * _ms,
		3 * _ms,
		4 * _ms,
		5 * _ms,
		6 * _ms,
		7 * _ms,
		8 * _ms,
		9 * _ms,
		10 * _ms,
		12 * _ms,
		14 * _ms,
		16 * _ms,
		18 * _ms,
		20 * _ms,
		25 * _ms,
		30 * _ms,
		35 * _ms,
		40 * _ms,
		45 * _ms,
		50 * _ms,
		60 * _ms,
		70 * _ms,
		80 * _ms,
		90 * _ms,
		100 * _ms,
		120 * _ms,
		140 * _ms,
		160 * _ms,
		180 * _ms,
		200 * _ms,
		250 * _ms,
		300 * _ms,
		350 * _ms,
		400 * _ms,
		450 * _ms,
		500 * _ms,
		600 * _ms,
		700 * _ms,
		800 * _ms,
		900 * _ms,
		1000 * _ms,
		1500 * _ms,
		2000 * _ms,
		2500 * _ms,
		3000 * _ms,
		4000 * _ms,
		5000 * _ms,
		7500 * _ms,
		10000 * _ms,
	}
)
