// Code generated by "stringer -type=State ./pkg/lifecycle"; DO NOT EDIT.

// Copyright (c) 2025 Uber Technologies, Inc.
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

package lifecycle

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Idle-0]
	_ = x[Starting-1]
	_ = x[Running-2]
	_ = x[Stopping-3]
	_ = x[Stopped-4]
	_ = x[Errored-5]
}

const _State_name = "IdleStartingRunningStoppingStoppedErrored"

var _State_index = [...]uint8{0, 4, 12, 19, 27, 34, 41}

func (i State) String() string {
	if i < 0 || i >= State(len(_State_index)-1) {
		return "State(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _State_name[_State_index[i]:_State_index[i+1]]
}
