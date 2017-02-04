package sync

import (
	"errors"
	"fmt"
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
			msg: "Stop",
			actions: []LifecycleAction{
				StartAction{ExpectedState: Running},
				StopAction{ExpectedState: Stopped},
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
						StopAction{ExpectedState: Stopped, Wait: 20 * time.Millisecond},
						GetStateAction{ExpectedState: Stopping},
					},
					Wait: 10 * time.Millisecond,
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
			msg: "Overlapping start after stop",
			// ms: timeline
			// 00: 0: start.............starting
			// 20: |  1: stop
			// 40: X..|.................running
			// "":    | (wait 20)       stopping
			// 60:    X.................stopped
			// 80:       2: start
			//           X
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StartAction{
							Wait:          40 * time.Millisecond,
							ExpectedState: Running,
						},
						StopAction{
							Wait:          20 * time.Millisecond,
							ExpectedState: Stopped,
						},
						StartAction{
							Wait:          20 * time.Millisecond,
							ExpectedState: Stopped,
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
			// 00: 0: start.............starting
			// 10: |  1: start
			// 20: |  |  2: stop
			// 30: |  |  | 3: start
			// 40: |  |  | |  4: stop
			// 40: X  X  | X  |..........running
			//           |    |..........stopping
			// 60:       X    X..........stopped
			actions: []LifecycleAction{
				ConcurrentAction{
					Actions: []LifecycleAction{
						StartAction{
							Wait:          40 * time.Millisecond,
							ExpectedState: Running,
						},
						StartAction{
							Wait:          40 * time.Millisecond,
							ExpectedState: Running,
						},
						StopAction{
							Wait:          40 * time.Millisecond,
							ExpectedState: Stopped,
						},
						StartAction{
							Wait:          40 * time.Millisecond,
							ExpectedState: Running,
						},
						StopAction{
							Wait:          40 * time.Millisecond,
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
