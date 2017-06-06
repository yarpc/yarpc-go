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

package backoff

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExponential(t *testing.T) {
	type backoffAttempt struct {
		msg            string
		giveAttempt    uint
		giveRandResult int64
		wantBackoff    time.Duration
	}
	type testStruct struct {
		msg string

		giveBase time.Duration
		giveMin  time.Duration
		giveMax  time.Duration

		attempts []backoffAttempt

		wantErrors []string
	}
	tests := []testStruct{
		{
			msg:      "invalid base",
			giveBase: time.Duration(0),
			giveMax:  time.Duration(0),
			giveMin:  time.Duration(0),
			wantErrors: []string{
				"invalid base for exponential backoff, need greater than zero",
			},
		},
		{
			msg:      "invalid min",
			giveBase: time.Duration(1000),
			giveMax:  time.Duration(0),
			giveMin:  time.Duration(-100),
			wantErrors: []string{
				"invalid min for exponential backoff, need greater than or equal to zero",
			},
		},
		{
			msg:      "invalid max & min",
			giveBase: time.Duration(1000),
			giveMax:  time.Duration(-1),
			giveMin:  time.Duration(-100),
			wantErrors: []string{
				"invalid min for exponential backoff, need greater than or equal to zero",
				"invalid max for exponential backoff, need greater than or equal to zero",
			},
		},
		{
			msg:      "invalid with max less than min",
			giveBase: time.Duration(1000),
			giveMax:  time.Millisecond,
			giveMin:  time.Second,
			wantErrors: []string{
				"exponential max value must be greater than min value",
			},
		},
		{
			msg:      "valid durations",
			giveBase: time.Nanosecond,
			giveMax:  time.Nanosecond * 100,
			giveMin:  time.Duration(0),
			attempts: []backoffAttempt{
				{
					msg:            "zero attempt max backoff",
					giveAttempt:    0,
					giveRandResult: int64(1 << 0),
					wantBackoff:    time.Nanosecond,
				},
				{
					msg:            "zero attempt min backoff",
					giveAttempt:    0,
					giveRandResult: 0,
					wantBackoff:    time.Duration(0),
				},
				{
					msg:            "zero attempt min backoff (with wrapped rand value)",
					giveAttempt:    0,
					giveRandResult: int64(1<<0) + 1,
					wantBackoff:    time.Duration(0),
				},
				{
					msg:            "one attempt max backoff",
					giveAttempt:    1,
					giveRandResult: int64(1 << 1),
					wantBackoff:    time.Nanosecond * 2,
				},
				{
					msg:            "two attempts max backoff",
					giveAttempt:    2,
					giveRandResult: int64(1 << 2),
					wantBackoff:    time.Duration(int64(1 << 2)),
				},
				{
					msg:            "three attempts max backoff",
					giveAttempt:    3,
					giveRandResult: int64(1 << 3),
					wantBackoff:    time.Duration(int64(1 << 3)),
				},
				{
					msg:            "four attempts max backoff",
					giveAttempt:    4,
					giveRandResult: int64(1 << 4),
					wantBackoff:    time.Duration(int64(1 << 4)),
				},
				{
					msg:            "four attempts min backoff (with wrapped rand value)",
					giveAttempt:    4,
					giveRandResult: int64(1<<4) + 1,
					wantBackoff:    time.Duration(0),
				},
				{
					msg:            "attempts range higher than max value",
					giveAttempt:    30,
					giveRandResult: 100,
					wantBackoff:    time.Nanosecond * 100,
				},
				{
					msg:            "attempts range higher than max value (with wrapped rand value)",
					giveAttempt:    30,
					giveRandResult: 100 + 1,
					wantBackoff:    time.Duration(0),
				},
				{
					msg:            "attempts that cause overflows should go to max",
					giveAttempt:    63, // 1<<63 == -9223372036854775808
					giveRandResult: 100,
					wantBackoff:    time.Nanosecond * 100,
				},
				{
					msg:            "attempts that cause overflows should go to max (with wrapped rand)",
					giveAttempt:    63, // 1<<63 == -9223372036854775808
					giveRandResult: 100 + 1,
					wantBackoff:    time.Duration(0),
				},
				{
					msg:            "attempts that go beyond overflows should go to max",
					giveAttempt:    64, // 1<<64 == 0
					giveRandResult: 100,
					wantBackoff:    time.Nanosecond * 100,
				},
				{
					msg:            "attempts that go beyond overflows should go to max (with wrapped rand)",
					giveAttempt:    64, // 1<<64 == 0
					giveRandResult: 100 + 1,
					wantBackoff:    time.Duration(0),
				},
				{
					msg:            "max value with a random value that i choose",
					giveAttempt:    14,
					giveRandResult: 68,
					wantBackoff:    time.Duration(68),
				},
			},
		},
		{
			msg:      "valid duration with min",
			giveBase: time.Nanosecond,
			giveMax:  time.Nanosecond * 100,
			giveMin:  time.Nanosecond * 10,
			attempts: []backoffAttempt{
				{
					msg:            "zero attempt max backoff",
					giveAttempt:    0,
					giveRandResult: int64(1 << 0),
					wantBackoff:    time.Nanosecond * 11,
				},
				{
					msg:            "zero attempt min backoff",
					giveAttempt:    0,
					giveRandResult: 0,
					wantBackoff:    time.Nanosecond * 10,
				},
				{
					msg:            "one attempt max backoff",
					giveAttempt:    1,
					giveRandResult: int64(1 << 1),
					wantBackoff:    time.Nanosecond * 12,
				},
				{
					msg:            "two attempts max backoff",
					giveAttempt:    2,
					giveRandResult: int64(1 << 2),
					wantBackoff:    time.Nanosecond * 14,
				},
				{
					msg:            "three attempts max backoff",
					giveAttempt:    3,
					giveRandResult: int64(1 << 3),
					wantBackoff:    time.Nanosecond * 18,
				},
				{
					msg:            "four attempts max backoff",
					giveAttempt:    4,
					giveRandResult: int64(1 << 4),
					wantBackoff:    time.Nanosecond * 26,
				},
				{
					msg:            "four attempts min backoff (with wrapped rand value)",
					giveAttempt:    4,
					giveRandResult: int64(1<<4) + 1,
					wantBackoff:    time.Nanosecond * 10,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			randSrc := &mutableRandSrc{val: 0}
			exp, err := NewExponential(
				BaseJump(tt.giveBase),
				MinBackoff(tt.giveMin),
				MaxBackoff(tt.giveMax),
				randGenerator(rand.New(randSrc)),
			)
			if err != nil {
				assert.True(t, len(tt.wantErrors) > 0, "got unexpected error: %s", err.Error())
				for _, wantErr := range tt.wantErrors {
					assert.Contains(t, err.Error(), wantErr)
				}
				return
			}
			require.True(t, len(tt.wantErrors) == 0, "didn't get expected error")
			for _, attempt := range tt.attempts {
				randSrc.val = attempt.giveRandResult
				assert.Equal(t, attempt.wantBackoff, exp.Duration(attempt.giveAttempt), "backoff for backoffAttempt %q did not match", attempt.msg)
			}
		})
	}
}

// mutableRandSrc implements the rand.Source interface so we can get our random
// number generator to return whatever we want.
type mutableRandSrc struct {
	val int64
}

func (r *mutableRandSrc) Int63() int64 {
	return r.val
}

func (*mutableRandSrc) Seed(int64) {}
