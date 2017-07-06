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
	"testing"
	"time"

	"github.com/uber-go/tally"
	"go.uber.org/yarpc/api/backoff"
	"go.uber.org/yarpc/api/transport"
	iioutil "go.uber.org/yarpc/internal/ioutil"
	"go.uber.org/yarpc/internal/testtime"
	. "go.uber.org/yarpc/internal/yarpctest/outboundtest"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestMiddleware(t *testing.T) {
	type testStruct struct {
		msg string

		policyProvider PolicyProvider

		actions []MiddlewareAction
	}
	tests := []testStruct{
		{
			msg: "no retry",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(testtime.Millisecond*500),
				),
			).provider,
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
			msg:            "nil policy",
			policyProvider: newPolicyProviderBuilder().setDefault(nil).provider,
			actions: []MiddlewareAction{
				RequestAction{
					request: &transport.Request{
						Service:   "serv",
						Procedure: "proc",
						Body:      bytes.NewBufferString("body"),
					},
					reqTimeout: testtime.Second * 5,
					events: []*OutboundEvent{
						{
							WantTimeout:       testtime.Second * 5,
							WantTimeoutBounds: testtime.Second,
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
			msg: "single retry",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(testtime.Millisecond*500),
				),
			).provider,
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
							GiveError:     yarpcerrors.InternalErrorf("unknown error"),
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
			msg: "multiple retries",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(4),
					MaxRequestTimeout(testtime.Millisecond*500),
				),
			).provider,
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
							GiveError:     yarpcerrors.InternalErrorf("unknown error"),
						},
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*300),
						},
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     yarpcerrors.DeadlineExceededErrorf("remote timed out"),
						},
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     yarpcerrors.InternalErrorf("unknown error"),
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
			msg: "immediate hard failure",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(testtime.Millisecond*500),
				),
			).provider,
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
							GiveError:     yarpcerrors.InvalidArgumentErrorf("bad request!"),
						},
					},
					wantError: yarpcerrors.InvalidArgumentErrorf("bad request!").Error(),
				},
			},
		},
		{
			msg: "retry once, then hard failure",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(testtime.Millisecond*500),
				),
			).provider,
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
							GiveError:     yarpcerrors.InternalErrorf("unknown error"),
						},
						{
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     yarpcerrors.InvalidArgumentErrorf("bad request!"),
						},
					},
					wantError: yarpcerrors.InvalidArgumentErrorf("bad request!").Error(),
				},
			},
		},
		{
			msg: "ctx timeout less than retry timeout",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(testtime.Millisecond*500),
				),
			).provider,
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
			msg: "ctx timeout less than retry timeout",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(testtime.Millisecond*50),
				),
			).provider,
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
							GiveError:      yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*50),
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
			msg: "no ctx timeout",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(testtime.Millisecond*50),
				),
			).provider,
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
							GiveError:      yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*50),
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
			msg: "exhaust retries",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(testtime.Millisecond*50),
				),
			).provider,
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
							GiveError:     yarpcerrors.InternalErrorf("unexpected error 1"),
						},
						{
							WantTimeout:   testtime.Millisecond * 50,
							WantService:   "serv",
							WantProcedure: "proc",
							WantBody:      "body",
							GiveError:     yarpcerrors.InternalErrorf("unexpected error 2"),
						},
					},
					wantError: yarpcerrors.InternalErrorf("unexpected error 2").Error(),
				},
			},
		},
		{
			msg: "Reset Error",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(testtime.Millisecond*50),
				),
			).provider,
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
							GiveError: yarpcerrors.InternalErrorf("unexpected error 1"),
						},
					},
					wantError: iioutil.ErrReset.Error(),
				},
			},
		},
		{
			msg: "backoff timeout",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(testtime.Millisecond*50),
					BackoffStrategy(newFixedBackoff(testtime.Millisecond*25)),
				),
			).provider,
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
							GiveError:      yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*50),
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
			msg: "sequential backoff timeout",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(2),
					MaxRequestTimeout(testtime.Millisecond*100),
					BackoffStrategy(newSequentialBackoff(testtime.Millisecond*50)),
				),
			).provider,
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
							GiveError:         yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*50),
						},
						{
							WantTimeout:       testtime.Millisecond * 100,
							WantTimeoutBounds: testtime.Millisecond * 20,
							WantService:       "serv",
							WantProcedure:     "proc",
							WantBody:          "body",
							WaitForTimeout:    true,
							GiveError:         yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*50),
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
			msg: "backoff context will timeout",
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(2),
					MaxRequestTimeout(testtime.Millisecond*30),
					BackoffStrategy(newFixedBackoff(testtime.Millisecond*5000)),
				),
			).provider,
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
							GiveError:         yarpcerrors.InternalErrorf("unexpected error 2"),
						},
					},
					wantTimeLimit: testtime.Millisecond * 40,
					wantError:     yarpcerrors.InternalErrorf("unexpected error 2").Error(),
				},
			},
		},
		{
			msg: "concurrent retries",
			// Policies:
			//   default: 	   retries=2   timeout=50ms  backoff=25ms
			// Request 1: "One retry"
			// ms :  timeout:100ms
			// 000:  initial request
			// 050:  error: Timeout
			// 075:  retry #1 (expected timeout for request: 25ms)
			// 075:  Successful response
			// Request 2: "Bad request does not retry"
			// ms :  timeout:1s
			// 000:  initial request
			// 050:  Final error: Bad Request (no retries)
			// Request 3: "Request times out"
			// ms :  timeout:100ms
			// 000:  initial request
			// 050:  error: Timeout
			// 075:  retry #1 (expected timeout for request: 25ms)
			// 100:  final error: Timeout
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(2),
					MaxRequestTimeout(testtime.Millisecond*50),
					BackoffStrategy(newFixedBackoff(testtime.Millisecond*25)),
				),
			).provider,
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
									GiveError:      yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*50),
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
									GiveError:     yarpcerrors.InvalidArgumentErrorf("bad request!"),
								},
							},
							wantError: yarpcerrors.InvalidArgumentErrorf("bad request!").Error(),
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
									GiveError:      yarpcerrors.DeadlineExceededErrorf("service:serv3 procedure:proc3 ttl:%v", testtime.Millisecond*50),
								},
								{
									WantTimeout:    testtime.Millisecond * 25,
									WantService:    "serv3",
									WantProcedure:  "proc3",
									WantBody:       "body3",
									GiveRespBody:   "respbody",
									WaitForTimeout: true,
									GiveError:      yarpcerrors.DeadlineExceededErrorf("service:serv3 procedure:proc3 ttl:%v", testtime.Millisecond*25),
								},
							},
							wantError: yarpcerrors.DeadlineExceededErrorf("service:serv3 procedure:proc3 ttl:%v", testtime.Millisecond*25).Error(),
						},
					},
				},
			},
		},
		{
			msg: "multiple retry policies",
			// Policies:
			//   default: 	   retries=2   timeout=20ms  backoff=25ms
			//   s="s": 	   retries=1   timeout=50ms  backoff=50ms
			//   s="s",p="p":  retries=0   timeout=100ms backoff=None
			//   s="s",p="fp": retries=100 timeout=10s   backoff=None // Fake policy
			// Request 1: "Default retry policy"
			// ms :  service:"ns" proc:"np" timeout:200ms expectedPolicy:"default"
			// 000:  initial request
			// 020:  error: Timeout
			// 045:  retry #1
			// 065:  error: Timeout
			// 090:  retry #2
			// 110:  final error: Timeout
			// Request 2: "Service Retry Policy"
			// ms :  service:"s" proc:"np" timeout:200ms expectedPolicy:"s="s""
			// 000:  initial request
			// 050:  error: Timeout
			// 100:  retry #1
			// 150:  final error: Timeout
			// Request 2: "ServiceProcedure Retry Policy"
			// ms :  service:"s" proc:"p" timeout:200ms expectedPolicy:"s="s",p="p""
			// 000:  initial request
			// 100:  final error: Timeout
			policyProvider: newPolicyProviderBuilder().setDefault(
				NewPolicy(
					Retries(2),
					MaxRequestTimeout(testtime.Millisecond*20),
					BackoffStrategy(newFixedBackoff(testtime.Millisecond*25)),
				),
			).registerService(
				"s",
				NewPolicy(
					Retries(1),
					MaxRequestTimeout(testtime.Millisecond*50),
					BackoffStrategy(newFixedBackoff(testtime.Millisecond*50)),
				),
			).registerServiceProcedure(
				"s",
				"p",
				NewPolicy(
					Retries(0),
					MaxRequestTimeout(testtime.Millisecond*100),
				),
			).registerServiceProcedure(
				"s",
				"fp",
				NewPolicy(
					Retries(100),
					MaxRequestTimeout(testtime.Millisecond*10000),
				),
			).provider,
			actions: []MiddlewareAction{
				ConcurrentAction{
					Actions: []MiddlewareAction{
						RequestAction{
							request: &transport.Request{
								Service:   "ns",
								Procedure: "np",
								Body:      bytes.NewBufferString("body1"),
							},
							reqTimeout: testtime.Millisecond * 200,
							events: []*OutboundEvent{
								{
									WantTimeout:    testtime.Millisecond * 20,
									WantService:    "ns",
									WantProcedure:  "np",
									WantBody:       "body1",
									WaitForTimeout: true,
									GiveError:      yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*20),
								},
								{
									WantTimeout:    testtime.Millisecond * 20,
									WantService:    "ns",
									WantProcedure:  "np",
									WantBody:       "body1",
									WaitForTimeout: true,
									GiveError:      yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*20),
								},
								{
									WantTimeout:   testtime.Millisecond * 20,
									WantService:   "ns",
									WantProcedure: "np",
									WantBody:      "body1",
									GiveError:     yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*20),
								},
							},
							wantError: yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*20).Error(),
						},
						RequestAction{
							request: &transport.Request{
								Service:   "s",
								Procedure: "np",
								Body:      bytes.NewBufferString("body2"),
							},
							reqTimeout: testtime.Millisecond * 200,
							events: []*OutboundEvent{
								{
									WantTimeout:    testtime.Millisecond * 50,
									WantService:    "s",
									WantProcedure:  "np",
									WantBody:       "body2",
									WaitForTimeout: true,
									GiveError:      yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*50),
								},
								{
									WantTimeout:   testtime.Millisecond * 50,
									WantService:   "s",
									WantProcedure: "np",
									WantBody:      "body2",
									GiveError:     yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*50),
								},
							},
							wantError: yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*50).Error(),
						},
						RequestAction{
							request: &transport.Request{
								Service:   "s",
								Procedure: "p",
								Body:      bytes.NewBufferString("body3"),
							},
							reqTimeout: testtime.Millisecond * 200,
							events: []*OutboundEvent{
								{
									WantTimeout:   testtime.Millisecond * 100,
									WantService:   "s",
									WantProcedure: "p",
									WantBody:      "body3",
									GiveError:     yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*100),
								},
							},
							wantError: yarpcerrors.DeadlineExceededErrorf("service:serv procedure:proc ttl:%v", testtime.Millisecond*100).Error(),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			retry := NewUnaryMiddleware(
				WithPolicyProvider(tt.policyProvider),
				TallyScope(tally.NoopScope),
			)

			ApplyMiddlewareActions(t, retry, tt.actions)
		})
	}
}

func TestNilRetry(t *testing.T) {
	mw := (*OutboundMiddleware)(nil)
	actions := []MiddlewareAction{
		RequestAction{
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			reqTimeout: testtime.Second * 5,
			events: []*OutboundEvent{
				{
					WantTimeout:       testtime.Second * 5,
					WantTimeoutBounds: testtime.Second,
					WantService:       "serv",
					WantProcedure:     "proc",
					WantBody:          "body",
					GiveRespBody:      "respbody",
				},
			},
			wantBody: "respbody",
		},
	}

	ApplyMiddlewareActions(t, mw, actions)
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

type policyProviderBuilder struct {
	provider *ProcedurePolicyProvider
}

func newPolicyProviderBuilder() *policyProviderBuilder {
	return &policyProviderBuilder{
		provider: NewProcedurePolicyProvider(),
	}
}

func (pb *policyProviderBuilder) registerServiceProcedure(service, procedure string, pol *Policy) *policyProviderBuilder {
	pb.provider.RegisterServiceProcedure(service, procedure, pol)
	return pb
}

func (pb *policyProviderBuilder) registerService(service string, pol *Policy) *policyProviderBuilder {
	pb.provider.RegisterService(service, pol)
	return pb
}

func (pb *policyProviderBuilder) setDefault(pol *Policy) *policyProviderBuilder {
	pb.provider.SetDefault(pol)
	return pb
}
