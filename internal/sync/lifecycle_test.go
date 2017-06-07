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

package sync

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestLifecycleOnce(t *testing.T) {
	type testStruct struct {
		msg string

		// A list of actions that will be applied on the LifecycleOnce
		actions []LifecycleAction

		// expected state at the end of the actions
		expectedFinalState LifecycleState
	}
	tests := []testStruct{
		{
			msg:                "setup",
			expectedFinalState: Idle,
		},
		{
			msg: "Start",
			actions: []LifecycleAction{
				StartAction{ExpectedState: Running},
			},
			expectedFinalState: Running,
		},
		{
			msg: "Start and Started",
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StartAction{ExpectedState: Running},
						Actions{
							WaitForStartAction,
							GetStateAction{ExpectedState: Running},
						},
					},
				},
			},
			expectedFinalState: Running,
		},
		{
			msg: "Stop",
			actions: []LifecycleAction{
				StartAction{ExpectedState: Running},
				StopAction{ExpectedState: Stopped},
			},
			expectedFinalState: Stopped,
		},
		{
			msg: "Stop and Stopped",
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StopAction{ExpectedState: Stopped},
						WaitForStoppingAction,
						WaitForStopAction,
					},
				},
			},
			expectedFinalState: Stopped,
		},
		{
			msg: "Error and Stopped",
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StartAction{
							Err:           fmt.Errorf("abort"),
							ExpectedErr:   fmt.Errorf("abort"),
							ExpectedState: Errored,
						},
						Actions{
							WaitForStoppingAction,
							GetStateAction{ExpectedState: Errored},
						},
						Actions{
							WaitForStopAction,
							GetStateAction{ExpectedState: Errored},
						},
					},
				},
			},
			expectedFinalState: Errored,
		},
		{
			msg: "Start, Stop, and Stopped",
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						Actions{
							StartAction{ExpectedState: Running},
							StopAction{ExpectedState: Stopped},
						},
						WaitForStoppingAction,
						WaitForStopAction,
					},
				},
			},
			expectedFinalState: Stopped,
		},
		{
			msg: "Starting",
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StartAction{ExpectedState: Running, Wait: 20 * time.Millisecond},
						GetStateAction{ExpectedState: Starting},
					},
					Wait: 10 * time.Millisecond,
				},
			},
			expectedFinalState: Running,
		},
		{
			msg: "Stopping",
			actions: []LifecycleAction{
				StartAction{ExpectedState: Running},
				ConcurrentAction{
					Actions: []LifecycleAction{
						StopAction{ExpectedState: Stopped, Wait: 15 * time.Millisecond},
						GetStateAction{ExpectedState: Stopping},
						GetStateAction{ExpectedState: Stopped},
					},
					Wait: 10 * time.Millisecond,
				},
			},
			expectedFinalState: Stopped,
		},
		{
			msg: "Delayed stop, wait for stopping",
			actions: []LifecycleAction{
				StartAction{ExpectedState: Running},
				ConcurrentAction{
					Actions: []LifecycleAction{
						Actions{
							StartAction{ExpectedState: Running},
							WaitAction(20 * time.Millisecond),
							StopAction{ExpectedState: Stopped, Wait: 20 * time.Millisecond},
						},
						Actions{
							WaitForStoppingAction,
							WaitAction(10 * time.Millisecond),
							ExactStateAction{ExpectedState: Stopping},
							WaitAction(20 * time.Millisecond),
							GetStateAction{ExpectedState: Stopped},
						},
					},
				},
			},
			expectedFinalState: Stopped,
		},
		{
			msg: "Start assure only called once and propagates the same error",
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StartAction{
							Wait:          40 * time.Millisecond,
							Err:           errors.New("expected error"),
							ExpectedState: Errored,
							ExpectedErr:   errors.New("expected error"),
						},
						StartAction{
							Err:           errors.New("not an expected error 1"),
							ExpectedState: Errored,
							ExpectedErr:   errors.New("expected error"),
						},
						StartAction{
							Wait:          40 * time.Millisecond,
							Err:           errors.New("not an expected error 2"),
							ExpectedState: Errored,
							ExpectedErr:   errors.New("expected error"),
						},
					},
					Wait: 10 * time.Millisecond,
				},
			},
			expectedFinalState: Errored,
		},
		{
			msg: "Successful Start followed by failed Stop",
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StartAction{
							Wait:          10 * time.Millisecond,
							ExpectedState: Running,
						},
						StopAction{
							Wait:          10 * time.Millisecond,
							Err:           errors.New("expected error"),
							ExpectedState: Errored,
							ExpectedErr:   errors.New("expected error"),
						},
						StartAction{
							Err:           errors.New("not expected error 2"),
							ExpectedState: Errored,
							ExpectedErr:   errors.New("expected error"),
						},
						StopAction{
							Err:           errors.New("not expected error 2"),
							ExpectedState: Errored,
							ExpectedErr:   errors.New("expected error"),
						},
					},
					Wait: 30 * time.Millisecond,
				},
			},
			expectedFinalState: Errored,
		},
		{
			msg: "Stop assure only called once and returns the same error",
			actions: []LifecycleAction{
				StartAction{
					ExpectedState: Running,
				},
				ConcurrentAction{
					Actions: []LifecycleAction{
						StopAction{
							Wait:          40 * time.Millisecond,
							Err:           errors.New("expected error"),
							ExpectedState: Errored,
							ExpectedErr:   errors.New("expected error"),
						},
						StopAction{
							Wait:          40 * time.Millisecond,
							Err:           errors.New("not an expected error 1"),
							ExpectedState: Errored,
							ExpectedErr:   errors.New("expected error"),
						},
						StopAction{
							Wait:          40 * time.Millisecond,
							Err:           errors.New("not an expected error 2"),
							ExpectedState: Errored,
							ExpectedErr:   errors.New("expected error"),
						},
					},
					Wait: 10 * time.Millisecond,
				},
			},
			expectedFinalState: Errored,
		},
		{
			msg: "Stop before start goes directly to 'stopped'",
			actions: []LifecycleAction{
				StopAction{
					ExpectedState: Stopped,
				},
			},
			expectedFinalState: Stopped,
		},
		{
			msg: "Pre-empting start after stop",
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StopAction{
							Wait:          10 * time.Millisecond,
							ExpectedState: Stopped,
						},
						StartAction{
							Err:           fmt.Errorf("start action should not run"),
							Wait:          500 * time.Second,
							ExpectedState: Stopped,
						},
					},
					Wait: 20 * time.Millisecond,
				},
			},
			expectedFinalState: Stopped,
		},
		{
			msg: "Overlapping stop after start",
			// ms: timeline
			// 00: 0: start..............starting
			// 10: |  1. stop
			// 50: X..|..................running
			//        |..................stopping
			// 60:    X..................stopped
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StartAction{
							Wait:          50 * time.Millisecond,
							ExpectedState: Running,
						},
						StopAction{
							Wait:          10 * time.Millisecond,
							ExpectedState: Stopped,
						},
					},
					Wait: 10 * time.Millisecond,
				},
			},
			expectedFinalState: Stopped,
		},
		{
			msg: "Overlapping stop after start error",
			// ms: timeline
			// 00: 0: start..............starting
			// 10: |  1. stop............stopping
			// 50: X  X..................errored
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StartAction{
							Wait:          50 * time.Millisecond,
							Err:           errors.New("expected error"),
							ExpectedState: Errored,
							ExpectedErr:   errors.New("expected error"),
						},
						StopAction{
							Wait:          10 * time.Millisecond,
							ExpectedState: Errored,
							ExpectedErr:   errors.New("expected error"),
						},
					},
					Wait: 10 * time.Millisecond,
				},
			},
			expectedFinalState: Errored,
		},
		{
			msg: "Overlapping start after stop",
			// ms: timeline
			// 00:  0: start.............starting
			// 10:  |
			// 20:  |  1: stop
			// 30:  X  -.................running
			// 30+Δ:   |.................stopping
			// 40:     |  2: start
			// 40+Δ:   |  X
			// 50:     |
			// 60:     |
			// 70:     X.................stopped
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StartAction{
							Wait:          30 * time.Millisecond,
							ExpectedState: Running,
						},
						StopAction{
							Wait:          30 * time.Millisecond,
							ExpectedState: Stopped,
						},
						StartAction{
							Err:           fmt.Errorf("start action should not run"),
							ExpectedState: Stopping,
						},
					},
					Wait: 20 * time.Millisecond,
				},
			},
			expectedFinalState: Stopped,
		},
		{
			msg: "Start completes before overlapping stop completes",
			// ms: timeline
			// 00:   0: start............starting
			// 10:   |  1: start
			// 20:   |  -  2: stop
			// 30:   X  -  -  3: start...running
			// 30+Δ:    X  |  X..........stopping
			// 40:         |     4: stop
			//             |     -
			//             |     -
			// 60:         X     X.......stopped
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StartAction{
							Wait:          30 * time.Millisecond,
							ExpectedState: Running,
						},
						StartAction{
							Err:           fmt.Errorf("start action should not run"),
							ExpectedState: Running,
						},
						StopAction{
							Wait:          40 * time.Millisecond,
							ExpectedState: Stopped,
						},
						StartAction{
							Err:           fmt.Errorf("start action should not run"),
							ExpectedState: Running,
						},
						StopAction{
							Err:           fmt.Errorf("stop action should not run"),
							ExpectedState: Stopped,
						},
					},
					Wait: 10 * time.Millisecond,
				},
			},
			expectedFinalState: Stopped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			once := Once()
			ApplyLifecycleActions(t, once, tt.actions)

			assert.Equal(t, tt.expectedFinalState, once.LifecycleState())
		})
	}
}

// TestStopping verifies that a lifecycle object can spawn a goroutine and wait
// for that goroutine to exit in its stopping state.  The goroutine must wrap
// up its work when it detects that the lifecycle has begun stopping.  If it
// waited for the stopped channel, the stop callback would deadlock.
func TestStopping(t *testing.T) {
	l := Once()
	l.Start(nil)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		<-l.Stopping()
		wg.Done()
	}()

	l.Stop(func() error {
		wg.Wait()
		return nil
	})
}
