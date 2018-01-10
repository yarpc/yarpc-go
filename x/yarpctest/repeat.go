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

package yarpctest

import (
	"fmt"
	"sync"
	"testing"

	"go.uber.org/yarpc/x/yarpctest/api"
)

// RepeatAction will call the provided action a set number of times (the action
// must be idempotent).
func RepeatAction(action Action, times int) api.Action {
	return api.ActionFunc(func(t testing.TB) {
		for i := 0; i < times; i++ {
			action.Run(t)
		}
	})
}

// ConcurrentAction will call the provided action in n concurrent threads at the
// same time.
func ConcurrentAction(action Action, threads int) api.Action {
	return api.ActionFunc(func(t testing.TB) {
		var wg sync.WaitGroup
		for i := 0; i < threads; i++ {
			wg.Add(1)
			go func(name string) {
				api.Run(
					name,
					t,
					func(t testing.TB) {
						action.Run(t)
					},
				)
				wg.Done()
			}(fmt.Sprint(i))
		}
		wg.Wait()
	})
}
