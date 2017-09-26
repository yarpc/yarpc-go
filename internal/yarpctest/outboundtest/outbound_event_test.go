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

package outboundtest

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/testtime"
	iyarpctest "go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/yarpctest"
)

func TestOutboundEvent(t *testing.T) {
	type testStruct struct {
		msg string

		request    *transport.Request
		reqTimeout time.Duration

		event *OutboundEvent

		wantExecutionStatus  iyarpctest.FakeTestStatus
		wantExecutionErrors  []string
		wantError            string
		wantApplicationError bool
		wantBody             string
		wantRespHeaders      transport.Headers
	}
	tests := []testStruct{
		{
			msg: "successful everything",
			request: &transport.Request{
				Caller:          "caller",
				Service:         "service",
				Encoding:        transport.Encoding("encoding"),
				Procedure:       "procedure",
				ShardKey:        "shard",
				RoutingKey:      "routekey",
				RoutingDelegate: "routedel",
				Features: transport.Features{
					SupportsBothResponseAndError: true,
				},
				Headers: transport.NewHeaders().With("key", "val"),
				Body:    bytes.NewBufferString("body"),
			},
			reqTimeout: testtime.Second,
			event: &OutboundEvent{
				WantTimeout:         testtime.Second,
				WantTimeoutBounds:   testtime.Millisecond * 20,
				WantCaller:          "caller",
				WantService:         "service",
				WantEncoding:        transport.Encoding("encoding"),
				WantProcedure:       "procedure",
				WantShardKey:        "shard",
				WantRoutingKey:      "routekey",
				WantRoutingDelegate: "routedel",
				WantFeatures: transport.Features{
					SupportsBothResponseAndError: true,
				},
				WantHeaders:  transport.NewHeaders().With("key", "val"),
				WantBody:     "body",
				GiveRespBody: "respbody",
			},
			wantExecutionStatus: iyarpctest.Finished,
			wantBody:            "respbody",
		},
		{
			msg: "errored request params",
			request: &transport.Request{
				Caller:          "caller2",
				Service:         "service2",
				Encoding:        transport.Encoding("encoding2"),
				Procedure:       "procedure2",
				ShardKey:        "shard2",
				RoutingKey:      "routekey2",
				RoutingDelegate: "routedel2",
				Features: transport.Features{
					SupportsBothResponseAndError: true,
				},
				Headers: transport.NewHeaders().With("key2", "val2"),
				Body:    bytes.NewBufferString("body"),
			},
			reqTimeout: testtime.Second,
			event: &OutboundEvent{
				WantCaller:          "caller",
				WantService:         "service",
				WantEncoding:        transport.Encoding("encoding"),
				WantProcedure:       "procedure",
				WantShardKey:        "shard",
				WantRoutingKey:      "routekey",
				WantRoutingDelegate: "routedel",
				WantFeatures: transport.Features{
					SupportsBothResponseAndError: true,
				},
				WantHeaders:  transport.NewHeaders().With("key", "val"),
				WantBody:     "body",
				GiveRespBody: "respbody",
			},
			wantBody:            "respbody",
			wantExecutionStatus: iyarpctest.Finished,
			wantExecutionErrors: []string{
				"invalid Caller",
				"invalid Service",
				"invalid Encoding",
				"invalid Procedure",
				"invalid ShardKey",
				"invalid RoutingKey",
				"invalid RoutingDelegate",
				"invalid Features",
				`header key "key" was not in request headers`,
				`invalid request header value for "key"`,
			},
		},
		{
			msg: "ignore extra request fields",
			request: &transport.Request{
				Caller:          "caller2",
				Service:         "service2",
				Encoding:        transport.Encoding("encoding2"),
				Procedure:       "procedure2",
				ShardKey:        "shard2",
				RoutingKey:      "routekey2",
				RoutingDelegate: "routedel2",
				Headers:         transport.NewHeaders().With("key2", "val2"),
				Body:            bytes.NewBufferString("body"),
			},
			reqTimeout: testtime.Second,
			event: &OutboundEvent{
				WantTimeout:       testtime.Second,
				WantTimeoutBounds: testtime.Millisecond * 20,
				WantBody:          "body",
				GiveRespBody:      "respbody",
			},
			wantBody:            "respbody",
			wantExecutionStatus: iyarpctest.Finished,
		},
		{
			msg: "default timeout range",
			request: &transport.Request{
				Body: bytes.NewBufferString("body"),
			},
			reqTimeout: testtime.Second,
			event: &OutboundEvent{
				WantTimeout:  testtime.Second,
				WantBody:     "body",
				GiveRespBody: "respbody",
			},
			wantBody:            "respbody",
			wantExecutionStatus: iyarpctest.Finished,
		},
		{
			msg: "timeout smaller than expected",
			request: &transport.Request{
				Body: bytes.NewBufferString("body"),
			},
			reqTimeout: testtime.Second,
			event: &OutboundEvent{
				WantTimeout:  testtime.Second * 2,
				WantBody:     "body",
				GiveRespBody: "respbody",
			},
			wantBody:            "respbody",
			wantExecutionStatus: iyarpctest.Finished,
			wantExecutionErrors: []string{
				"deadline was less than expected",
			},
		},
		{
			msg: "timeout larger than expected",
			request: &transport.Request{
				Body: bytes.NewBufferString("body"),
			},
			reqTimeout: testtime.Second * 2,
			event: &OutboundEvent{
				WantTimeout:  testtime.Second,
				WantBody:     "body",
				GiveRespBody: "respbody",
			},
			wantBody:            "respbody",
			wantExecutionStatus: iyarpctest.Finished,
			wantExecutionErrors: []string{
				"deadline was greater than expected",
			},
		},
		{
			msg: "wanttimeout with no deadline",
			request: &transport.Request{
				Body: bytes.NewBufferString("body"),
			},
			event: &OutboundEvent{
				WantTimeout:  testtime.Second,
				WantBody:     "body",
				GiveRespBody: "respbody",
			},
			wantExecutionStatus: iyarpctest.Fatal,
			wantExecutionErrors: []string{
				"wanted context deadline, but there was no deadline",
			},
		},
		{
			msg: "invalid number of header keys",
			request: &transport.Request{
				Headers: transport.NewHeaders().With("key2", "val2").With("key3", "val3"),
				Body:    bytes.NewBufferString("body"),
			},
			reqTimeout: testtime.Second,
			event: &OutboundEvent{
				WantHeaders:  transport.NewHeaders().With("key", "val"),
				WantBody:     "body",
				GiveRespBody: "respbody",
			},
			wantBody:            "respbody",
			wantExecutionStatus: iyarpctest.Finished,
			wantExecutionErrors: []string{
				"unexpected number of headers",
				`header key "key" was not in request headers`,
				`invalid request header value for "key"`,
			},
		},
		{
			msg: "invalid body",
			request: &transport.Request{
				Body: bytes.NewBufferString("body22"),
			},
			reqTimeout: testtime.Second,
			event: &OutboundEvent{
				WantBody:     "body",
				GiveRespBody: "respbody",
			},
			wantBody:            "respbody",
			wantExecutionStatus: iyarpctest.Finished,
			wantExecutionErrors: []string{
				"request body did not match",
			},
		},
		{
			msg: "wait for timeout",
			request: &transport.Request{
				Body: bytes.NewBufferString("body"),
			},
			reqTimeout: testtime.Millisecond * 10,
			event: &OutboundEvent{
				WaitForTimeout: true,
				WantBody:       "body",
				GiveRespBody:   "respbody",
			},
			wantBody:            "respbody",
			wantExecutionStatus: iyarpctest.Finished,
		},
		{
			msg: "wait for timeout with no deadline",
			request: &transport.Request{
				Body: bytes.NewBufferString("body"),
			},
			event: &OutboundEvent{
				WaitForTimeout: true,
				WantBody:       "body",
				GiveRespBody:   "respbody",
			},
			wantExecutionStatus: iyarpctest.Fatal,
			wantExecutionErrors: []string{
				"attempted to wait on context that has no deadline",
			},
		},
		{
			msg: "validate call responses",
			request: &transport.Request{
				Body: bytes.NewBufferString("body"),
			},
			event: &OutboundEvent{
				WantBody:             "body",
				GiveRespBody:         "respbody",
				GiveApplicationError: true,
				GiveError:            errors.New("test error"),
				GiveRespHeaders:      transport.NewHeaders().With("key", "val"),
			},
			wantBody:             "respbody",
			wantApplicationError: true,
			wantError:            "test error",
			wantRespHeaders:      transport.NewHeaders().With("key", "val"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			testResult := iyarpctest.WithFakeTestingT(func(ft require.TestingT) {
				ctx := context.Background()
				if tt.reqTimeout != 0 {
					newCtx, cancel := context.WithTimeout(ctx, tt.reqTimeout)
					defer cancel()
					ctx = newCtx
				}
				resp, err := tt.event.Call(ctx, ft, tt.request)

				if tt.wantError != "" {
					assert.EqualError(t, err, tt.wantError)
					require.NotNil(t, resp)
					assert.Equal(t, tt.wantApplicationError, resp.ApplicationError)
				} else {
					require.NotNil(t, resp)
					body, err := ioutil.ReadAll(resp.Body)
					assert.NoError(t, err)
					assert.Equal(t, tt.wantBody, string(body))
				}
				if tt.wantRespHeaders.Len() != 0 {
					assert.Equal(t, tt.wantRespHeaders.Len(), resp.Headers.Len(), "unexpected number of headers")
					for key, wantVal := range tt.wantRespHeaders.Items() {
						gotVal, ok := resp.Headers.Get(key)
						assert.True(t, ok, "header key %q was not in response headers", key)
						assert.Equal(t, wantVal, gotVal, "invalid response header value for %q", key)
					}
				}
			})
			assert.Equal(t, tt.wantExecutionStatus, testResult.Status)
			require.Equal(t, len(tt.wantExecutionErrors), len(testResult.Errors))
			for i, wantErr := range tt.wantExecutionErrors {
				assert.Contains(t, testResult.Errors[i], wantErr)
			}
		})
	}
}

func TestOutboundCallable(t *testing.T) {
	type testStruct struct {
		msg string

		reqs       []*transport.Request
		reqTimeout time.Duration

		events []*OutboundEvent

		wantExecutionStatus iyarpctest.FakeTestStatus
		wantExecutionErrors []string
	}
	tests := []testStruct{
		{
			msg: "equal calls",
			reqs: []*transport.Request{
				{
					Service:   "serv",
					Procedure: "proc",
					Body:      bytes.NewBufferString("body"),
				},
			},
			reqTimeout: time.Second,
			events: []*OutboundEvent{
				{
					WantService:   "serv",
					WantProcedure: "proc",
					WantBody:      "body",
				},
			},
			wantExecutionStatus: iyarpctest.Finished,
		},
		{
			msg: "extra call",
			reqs: []*transport.Request{
				{
					Service:   "serv",
					Procedure: "proc",
					Body:      bytes.NewBufferString("body"),
				},
				{
					Service:   "serv",
					Procedure: "proc",
					Body:      bytes.NewBufferString("body"),
				},
			},
			reqTimeout: time.Second,
			events: []*OutboundEvent{
				{
					WantService:   "serv",
					WantProcedure: "proc",
					WantBody:      "body",
				},
			},
			wantExecutionStatus: iyarpctest.Fatal,
			wantExecutionErrors: []string{
				"attempted to execute event #2 on the outbound, there are only 1 events",
				"did not execute the proper number of outbound calls",
			},
		},
		{
			msg: "not enough calls",
			reqs: []*transport.Request{
				{
					Service:   "serv",
					Procedure: "proc",
					Body:      bytes.NewBufferString("body"),
				},
			},
			reqTimeout: time.Second,
			events: []*OutboundEvent{
				{
					WantService:   "serv",
					WantProcedure: "proc",
					WantBody:      "body",
				},
				{
					WantService:   "serv",
					WantProcedure: "proc",
					WantBody:      "body",
				},
			},
			wantExecutionStatus: iyarpctest.Finished,
			wantExecutionErrors: []string{
				"did not execute the proper number of outbound calls",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			testResult := iyarpctest.WithFakeTestingT(func(ft require.TestingT) {
				callable := NewOutboundEventCallable(ft, tt.events)
				defer callable.Cleanup()

				trans := yarpctest.NewFakeTransport()
				out := trans.NewOutbound(yarpctest.NewFakePeerList(), yarpctest.OutboundCallOverride(callable.Call))
				out.Start()

				ctx := context.Background()
				if tt.reqTimeout != 0 {
					newCtx, cancel := context.WithTimeout(ctx, tt.reqTimeout)
					defer cancel()
					ctx = newCtx
				}

				for _, req := range tt.reqs {
					out.Call(ctx, req)
				}
			})
			assert.Equal(t, tt.wantExecutionStatus, testResult.Status)
			require.Equal(t, len(tt.wantExecutionErrors), len(testResult.Errors))
			for i, wantErr := range tt.wantExecutionErrors {
				assert.Contains(t, testResult.Errors[i], wantErr)
			}
		})
	}
}
