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

package backoff

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInvalidFirst(t *testing.T) {
	_, err := NewExponential(
		FirstBackoff(time.Duration(0)),
	)
	assert.Equal(t, err.Error(), "invalid first duration for exponential backoff, need greater than zero")
}

func TestInvalidMax(t *testing.T) {
	_, err := NewExponential(
		MaxBackoff(-1 * time.Second),
	)
	assert.Equal(t, err.Error(), "invalid max for exponential backoff, need greater than or equal to zero")
}

func TestInvalidFirstAndMax(t *testing.T) {
	_, err := NewExponential(
		FirstBackoff(time.Duration(0)),
		MaxBackoff(-1*time.Second),
	)
	assert.Equal(t, err.Error(), "invalid first duration for exponential backoff, need greater than zero; invalid max for exponential backoff, need greater than or equal to zero")
}

func TestExponential(t *testing.T) {
	type backoffAttempt struct {
		msg            string
		giveAttempt    uint
		giveRandResult int64
		wantBackoff    time.Duration
	}
	type testStruct struct {
		msg string

		giveFirst time.Duration
		giveMax   time.Duration

		attempts []backoffAttempt
	}
	tests := []testStruct{
		{
			msg:       "valid durations",
			giveFirst: time.Nanosecond,
			giveMax:   time.Nanosecond * 100,
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
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			randSrc := &mutableRandSrc{val: 0}
			strategy, err := NewExponential(
				FirstBackoff(tt.giveFirst),
				MaxBackoff(tt.giveMax),
				randGenerator(func() *rand.Rand { return rand.New(randSrc) }),
			)
			assert.NoError(t, err)
			backoff := strategy.Backoff()
			for _, attempt := range tt.attempts {
				randSrc.val = attempt.giveRandResult
				assert.Equal(t, attempt.wantBackoff, backoff.Duration(attempt.giveAttempt), "backoff for backoffAttempt %q did not match", attempt.msg)
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
