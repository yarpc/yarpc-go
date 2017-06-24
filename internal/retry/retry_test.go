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

package retry

import (
	"bytes"
	"context"
	"testing"
	"time"

	"go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
	iioutil "go.uber.org/yarpc/internal/ioutil"
	"go.uber.org/yarpc/internal/testtime"
	. "go.uber.org/yarpc/internal/yarpctest/outboundtest"
)

func TestMiddleware(t *testing.T) {
	type testStruct struct {
		msg string

		retries      uint
		retryTimeout time.Duration
		retryBackoff backoff.Strategy

		actions []MiddlewareAction
	}
	tests := []testStruct{
		{
			msg:          "no retry",
			retries:      1,
			retryTimeout: testtime.Millisecond * 500,
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Second,
					events: []*OutboundEvent{
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveRespBody:  "respbody",
						},
					},
					wantBody: "respbody",
				},
			},
		},
		{
			msg:          "single retry",
			retries:      1,
			retryTimeout: testtime.Millisecond * 500,
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Second,
					events: []*OutboundEvent{
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     errors.RemoteUnexpectedError("unknown error"),
						},
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveRespBody:  "respbody",
						},
					},
					wantBody: "respbody",
				},
			},
		},
		{
			msg:          "multiple retries",
			retries:      4,
			retryTimeout: testtime.Millisecond * 500,
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Second,
					events: []*OutboundEvent{
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     errors.RemoteUnexpectedError("unknown error"),
						},
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     errors.ClientTimeoutError("serv", "proc", testtime.Millisecond*300),
						},
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     errors.RemoteTimeoutError("remote timed out"),
						},
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     errors.RemoteUnexpectedError("unknown error"),
						},
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveRespBody:  "respbody",
						},
					},
					wantBody: "respbody",
				},
			},
		},
		{
			msg:          "immediate hard failure",
			retries:      1,
			retryTimeout: testtime.Millisecond * 500,
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Second,
					events: []*OutboundEvent{
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     errors.RemoteBadRequestError("bad request!"),
						},
					},
					wantError: errors.RemoteBadRequestError("bad request!").Error(),
				},
			},
		},
		{
			msg:          "retry once, then hard failure",
			retries:      1,
			retryTimeout: testtime.Millisecond * 500,
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Second,
					events: []*OutboundEvent{
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     errors.RemoteUnexpectedError("unknown error"),
						},
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     errors.RemoteBadRequestError("bad request!"),
						},
					},
					wantError: errors.RemoteBadRequestError("bad request!").Error(),
				},
			},
		},
		{
			msg:          "ctx timeout less than retry timeout",
			retries:      1,
			retryTimeout: testtime.Millisecond * 500,
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Millisecond * 300,
					events: []*OutboundEvent{
						{
							WantTimeout:   testtime.Millisecond * 300,
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveRespBody:  "respbody",
						},
					},
					wantBody: "respbody",
				},
			},
		},
		{
			msg:          "ctx timeout less than retry timeout",
			retries:      1,
			retryTimeout: testtime.Millisecond * 50,
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Millisecond * 75,
					events: []*OutboundEvent{
						{
							WantTimeout:    testtime.Millisecond * 50,
							WantService:    "serv",
							WantProcedure:  "proc",
							WantBody:       "body",
							WaitForTimeout: true,
							GiveError:      errors.ClientTimeoutError("serv", "proc", testtime.Millisecond*50),
						},
						{
							WantTimeout:   testtime.Millisecond * 25,
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveRespBody:  "respbody",
						},
					},
					wantBody: "respbody",
				},
			},
		},
		{
			msg:          "no ctx timeout",
			retries:      1,
			retryTimeout: testtime.Millisecond * 50,
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					events: []*OutboundEvent{
						{
							WantTimeout:    testtime.Millisecond * 50,
							WantService:    "serv",
							WantProcedure:  "proc",
							WantBody:       "body",
							WaitForTimeout: true,
							GiveError:      errors.ClientTimeoutError("serv", "proc", testtime.Millisecond*50),
						},
						{
							WantTimeout:   testtime.Millisecond * 50,
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveRespBody:  "respbody",
						},
					},
					wantBody: "respbody",
				},
			},
		},
		{
			msg:          "exhaust retries",
			retries:      1,
			retryTimeout: testtime.Millisecond * 50,
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Millisecond * 400,
					events: []*OutboundEvent{
						{
							WantTimeout:   testtime.Millisecond * 50,
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     errors.RemoteUnexpectedError("unexpected error 1"),
						},
						{
							WantTimeout:   testtime.Millisecond * 50,
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     errors.RemoteUnexpectedError("unexpected error 2"),
						},
					},
					wantError: errors.RemoteUnexpectedError("unexpected error 2").Error(),
				},
			},
		},
		{
			msg:          "Reset Error",
			retries:      1,
			retryTimeout: testtime.Millisecond * 50,
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Millisecond * 400,
					events: []*OutboundEvent{
						{
							WantTimeout:   testtime.Millisecond * 50,
							WantService:   "serv",
							WantProcedure: "proc",
							// We have explicitly not read the body, which will not exhaust the
							// req body io.Reader.
							GiveError: errors.RemoteUnexpectedError("unexpected error 1"),
						},
					},
					wantError: iioutil.ErrReset.Error(),
				},
			},
		},
		{
			msg:          "backoff timeout",
			retries:      1,
			retryTimeout: testtime.Millisecond * 50,
			retryBackoff: newFixedBackoff(testtime.Millisecond * 25),
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Millisecond * 100,
					events: []*OutboundEvent{
						{
							WantTimeout:    testtime.Millisecond * 50,
							WantService:    "serv",
							WantProcedure:  "proc",
							WantBody:       "body",
							WaitForTimeout: true,
							GiveError:      errors.ClientTimeoutError("serv", "proc", testtime.Millisecond*50),
						},
						{
							WantTimeout:   testtime.Millisecond * 25,
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveRespBody:  "respbody",
						},
					},
					wantBody: "respbody",
				},
			},
		},
		{
			msg:          "sequential backoff timeout",
			retries:      2,
			retryTimeout: testtime.Millisecond * 100,
			retryBackoff: newSequentialBackoff(testtime.Millisecond * 50),
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Millisecond * 400,
					events: []*OutboundEvent{
						{
							WantTimeout:       testtime.Millisecond * 100,
							WantTimeoutBounds: testtime.Millisecond * 20,
							WantService:       "serv",
							WantProcedure:     "proc",
							WantBody:          "body",
							WaitForTimeout:    true,
							GiveError:         errors.ClientTimeoutError("serv", "proc", testtime.Millisecond*50),
						},
						{
							WantTimeout:       testtime.Millisecond * 100,
							WantTimeoutBounds: testtime.Millisecond * 20,
							WantService:       "serv",
							WantProcedure:     "proc",
							WantBody:          "body",
							WaitForTimeout:    true,
							GiveError:         errors.ClientTimeoutError("serv", "proc", testtime.Millisecond*50),
						},
						{
							WantTimeout:       testtime.Millisecond * 50,
							WantTimeoutBounds: testtime.Millisecond * 20,
							WantService:       "serv",
							WantProcedure:     "proc",
							WantBody:          "body",
							GiveRespBody:      "respbody",
						},
					},
					wantBody: "respbody",
				},
			},
		},
		{
			msg:          "backoff context will timeout",
			retries:      2,
			retryTimeout: testtime.Millisecond * 30,
			retryBackoff: newFixedBackoff(testtime.Millisecond * 5000),
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Millisecond * 60,
					events: []*OutboundEvent{
						{
							WantTimeout:       testtime.Millisecond * 30,
							WantTimeoutBounds: testtime.Millisecond * 10,
							WantService:       "serv",
							WantProcedure:     "proc",
							WantBody:          "body",
							WaitForTimeout:    true,
							GiveError:         errors.RemoteUnexpectedError("unexpected error 2"),
						},
					},
					wantTimeLimit: testtime.Millisecond * 40,
					wantError:     errors.RemoteUnexpectedError("unexpected error 2").Error(),
				},
			},
		},
		{
			msg:          "concurrent retries",
			retries:      2,
			retryTimeout: testtime.Millisecond * 50,
			retryBackoff: newFixedBackoff(testtime.Millisecond * 25),
			actions: []MiddlewareAction{
				ConcurrentAction{
					Actions: []MiddlewareAction{
						RequestAction{
							request: &transport.Request{
								Service:   "serv",
								Procedure: "proc",
								Body:      bytes.NewBufferString("body"),
							},
							reqTimeout: testtime.Millisecond * 100,
							events: []*OutboundEvent{
								{
									WantTimeout:    testtime.Millisecond * 50,
									WantService:    "serv",
									WantProcedure:  "proc",
									WantBody:       "body",
									WaitForTimeout: true,
									GiveError:      errors.ClientTimeoutError("serv", "proc", testtime.Millisecond*50),
								},
								{
									WantTimeout:   testtime.Millisecond * 25,
									WantService:   "serv",
									WantProcedure: "proc",
									WantBody:      "body",
									GiveRespBody:  "respbody",
								},
							},
							wantBody: "respbody",
						},
						RequestAction{
							request: &transport.Request{
								Service:   "serv2",
								Procedure: "proc2",
								Body:      bytes.NewBufferString("body2"),
							},
							reqTimeout: testtime.Second,
							events: []*OutboundEvent{
								{
									WantTimeout:   testtime.Millisecond * 50,
									WantService:   "serv2",
									WantProcedure: "proc2",
									WantBody:      "body2",
									GiveError:     errors.RemoteBadRequestError("bad request!"),
								},
							},
							wantError: errors.RemoteBadRequestError("bad request!").Error(),
						},
						RequestAction{
							request: &transport.Request{
								Service:   "serv3",
								Procedure: "proc3",
								Body:      bytes.NewBufferString("body3"),
							},
							reqTimeout: testtime.Millisecond * 100,
							events: []*OutboundEvent{
								{
									WantTimeout:    testtime.Millisecond * 50,
									WantService:    "serv3",
									WantProcedure:  "proc3",
									WantBody:       "body3",
									WaitForTimeout: true,
									GiveError:      errors.ClientTimeoutError("serv3", "proc3", testtime.Millisecond*50),
								},
								{
									WantTimeout:    testtime.Millisecond * 25,
									WantService:    "serv3",
									WantProcedure:  "proc3",
									WantBody:       "body3",
									GiveRespBody:   "respbody",
									WaitForTimeout: true,
									GiveError:      errors.ClientTimeoutError("serv3", "proc3", testtime.Millisecond*25),
								},
							},
							wantError: errors.ClientTimeoutError("serv3", "proc3", testtime.Millisecond*25).Error(),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			retry := NewUnaryMiddleware(
				WithPolicyProvider(
					func(context.Context, *transport.Request) *Policy {
						return NewPolicy(
							Retries(tt.retries),
							MaxRequestTimeout(tt.retryTimeout),
							BackoffStrategy(tt.retryBackoff),
						)
					},
				),
			)

			ApplyMiddlewareActions(t, retry, tt.actions)
		})
	}
}

// Sequential backoff will increment the backoff sequentially based
// on the number of attempts.
// It is useful in tests so we have a reproducible backoff time.
func newSequentialBackoff(base time.Duration) *sequentialBackoff {
	return &sequentialBackoff{base}
}

type sequentialBackoff struct {
	base time.Duration
}

func (s *sequentialBackoff) Backoff() backoff.Backoff {
	return s
}

func (s *sequentialBackoff) Duration(attempts uint) time.Duration {
	return time.Duration(s.base.Nanoseconds() * int64(attempts+1))
}

// Fixed backoff will always return the same backoff.
func newFixedBackoff(boff time.Duration) *fixedBackoff {
	return &fixedBackoff{boff}
}

type fixedBackoff struct {
	boff time.Duration
}

func (f *fixedBackoff) Backoff() backoff.Backoff {
	return f
}

func (f *fixedBackoff) Duration(_ uint) time.Duration {
	return f.boff
}
