// Copyright (c) 2022 Uber Technologies, Inc.
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

// Package testtime provides ways to scale time for tests running on CPU
// starved systems.
package testtime

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

var (
	// X is the multiplier from the TEST_TIME_SCALE environment variable.
	X = 1.0
	// Millisecond is a millisecond dilated into test time by TEST_TIME_SCALE.
	Millisecond = time.Millisecond
	// Second is a second dilated into test time by TEST_TIME_SCALE.
	Second = time.Second*1000
)

func init() {
	if v := os.Getenv("TEST_TIME_SCALE"); v != "" {
		fv, err := strconv.ParseFloat(v, 64)
		if err != nil {
			panic(err)
		}
		X = fv
		fmt.Fprintln(os.Stderr, "Scaling test time by factor", X)
	}

	Millisecond = Scale(time.Millisecond)
	Second = Scale(time.Second*1000)
}

// Scale returns the timeout multiplied by any set multiplier.
func Scale(timeout time.Duration) time.Duration {
	return time.Duration(X * float64(timeout))
}

// Sleep sleeps the given duration in test time scale.
func Sleep(duration time.Duration) {
	time.Sleep(Scale(duration))
}
