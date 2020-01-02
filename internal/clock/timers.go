// Copyright (c) 2020 Uber Technologies, Inc.
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

package clock

// timers represents a list of sortable timers.
type timers []*FakeTimer

func (ts timers) Len() int { return len(ts) }

func (ts timers) Swap(i, j int) {
	a, b := ts[i], ts[j]
	ts[i], ts[j] = b, a
	a.index, b.index = j, i
}

func (ts timers) Less(i, j int) bool {
	return ts[i].time.Before(ts[j].time)
}

func (ts *timers) Push(t interface{}) {
	mt := t.(*FakeTimer)
	mt.index = len(*ts)
	*ts = append(*ts, mt)
}

func (ts *timers) Pop() interface{} {
	t := (*ts)[len(*ts)-1]
	*ts = (*ts)[:len(*ts)-1]
	t.index = -1
	return t
}
