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
	"io/ioutil"
	"testing"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
	iioutil "go.uber.org/yarpc/internal/ioutil"
	. "go.uber.org/yarpc/internal/yarpctest/outboundtest"
	"go.uber.org/yarpc/yarpctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/backoff"
)

func TestMiddleware(t *testing.T) {
	type testStruct struct {
		msg string

		request    *transport.Request
		reqTimeout time.Duration

		retries      uint
		retrytimeout time.Duration
		retryBackoff backoff.Strategy

		events []*OutboundEvent

		wantError            string
		wantApplicationError bool
		wantBody             string
	}
	tests := []testStruct{
		{
			msg: "no retry",
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			reqTimeout:   time.Second,
			retries:      1,
			retrytimeout: time.Millisecond * 500,
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
		{
			msg: "single retry",
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			reqTimeout:   time.Second,
			retries:      1,
			retrytimeout: time.Millisecond * 500,
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
		{
			msg: "multiple retries",
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			reqTimeout:   time.Second,
			retries:      4,
			retrytimeout: time.Millisecond * 500,
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
					GiveError:     errors.ClientTimeoutError("serv", "proc", time.Millisecond*300),
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
		{
			msg: "immediate hard failure",
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			reqTimeout:   time.Second,
			retries:      1,
			retrytimeout: time.Millisecond * 500,
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
		{
			msg: "retry once, then hard failure",
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			reqTimeout:   time.Second,
			retries:      1,
			retrytimeout: time.Millisecond * 500,
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
		{
			msg: "ctx timeout less than retry timeout",
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			reqTimeout:   time.Millisecond * 300,
			retries:      1,
			retrytimeout: time.Millisecond * 500,
			events: []*OutboundEvent{
				{
					WantTimeout:   time.Millisecond * 300,
					WantService:   "serv",
					WantProcedure: "proc",
					WantBody:      "body",
					GiveRespBody:  "respbody",
				},
			},
			wantBody: "respbody",
		},
		{
			msg: "ctx timeout less than retry timeout",
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			reqTimeout:   time.Millisecond * 75,
			retries:      1,
			retrytimeout: time.Millisecond * 50,
			events: []*OutboundEvent{
				{
					WantTimeout:    time.Millisecond * 50,
					WantService:    "serv",
					WantProcedure:  "proc",
					WantBody:       "body",
					WaitForTimeout: true,
					GiveError:      errors.ClientTimeoutError("serv", "proc", time.Millisecond*50),
				},
				{
					WantTimeout:   time.Millisecond * 25,
					WantService:   "serv",
					WantProcedure: "proc",
					WantBody:      "body",
					GiveRespBody:  "respbody",
				},
			},
			wantBody: "respbody",
		},
		{
			msg: "no ctx timeout",
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			retries:      1,
			retrytimeout: time.Millisecond * 50,
			events: []*OutboundEvent{
				{
					WantTimeout:    time.Millisecond * 50,
					WantService:    "serv",
					WantProcedure:  "proc",
					WantBody:       "body",
					WaitForTimeout: true,
					GiveError:      errors.ClientTimeoutError("serv", "proc", time.Millisecond*50),
				},
				{
					WantTimeout:   time.Millisecond * 50,
					WantService:   "serv",
					WantProcedure: "proc",
					WantBody:      "body",
					GiveRespBody:  "respbody",
				},
			},
			wantBody: "respbody",
		},
		{
			msg: "exhaust retries",
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			reqTimeout:   time.Millisecond * 400,
			retries:      1,
			retrytimeout: time.Millisecond * 50,
			events: []*OutboundEvent{
				{
					WantTimeout:   time.Millisecond * 50,
					WantService:   "serv",
					WantProcedure: "proc",
					WantBody:      "body",
					GiveError:     errors.RemoteUnexpectedError("unexpected error 1"),
				},
				{
					WantTimeout:   time.Millisecond * 50,
					WantService:   "serv",
					WantProcedure: "proc",
					WantBody:      "body",
					GiveError:     errors.RemoteUnexpectedError("unexpected error 2"),
				},
			},
			wantError: errors.RemoteUnexpectedError("unexpected error 2").Error(),
		},
		{
			msg: "Reset Error",
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			reqTimeout:   time.Millisecond * 400,
			retries:      1,
			retrytimeout: time.Millisecond * 50,
			events: []*OutboundEvent{
				{
					WantTimeout:   time.Millisecond * 50,
					WantService:   "serv",
					WantProcedure: "proc",
					// We have explicitly not read the body, which will not exhaust the
					// req body io.Reader.
					GiveError: errors.RemoteUnexpectedError("unexpected error 1"),
				},
			},
			wantError: iioutil.ErrReset.Error(),
		},
		{
			msg: "backoff timeout",
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			reqTimeout:   time.Millisecond * 100,
			retries:      1,
			retrytimeout: time.Millisecond * 50,
			retryBackoff: backoff.FixedBackoff(time.Millisecond * 25).Backoff,
			events: []*OutboundEvent{
				{
					WantTimeout:    time.Millisecond * 50,
					WantService:    "serv",
					WantProcedure:  "proc",
					WantBody:       "body",
					WaitForTimeout: true,
					GiveError:      errors.ClientTimeoutError("serv", "proc", time.Millisecond*50),
				},
				{
					WantTimeout:   time.Millisecond * 25,
					WantService:   "serv",
					WantProcedure: "proc",
					WantBody:      "body",
					GiveRespBody:  "respbody",
				},
			},
			wantBody: "respbody",
		},
		{
			msg: "sequential backoff timeout",
			request: &transport.Request{
				Service:   "serv",
				Procedure: "proc",
				Body:      bytes.NewBufferString("body"),
			},
			reqTimeout:   time.Millisecond * 400,
			retries:      2,
			retrytimeout: time.Millisecond * 100,
			retryBackoff: newSequentialBackoff(time.Millisecond * 50).Backoff,
			events: []*OutboundEvent{
				{
					WantTimeout:       time.Millisecond * 100,
					WantTimeoutBounds: time.Millisecond * 20,
					WantService:       "serv",
					WantProcedure:     "proc",
					WantBody:          "body",
					WaitForTimeout:    true,
					GiveError:         errors.ClientTimeoutError("serv", "proc", time.Millisecond*50),
				},
				{
					WantTimeout:       time.Millisecond * 100,
					WantTimeoutBounds: time.Millisecond * 20,
					WantService:       "serv",
					WantProcedure:     "proc",
					WantBody:          "body",
					WaitForTimeout:    true,
					GiveError:         errors.ClientTimeoutError("serv", "proc", time.Millisecond*50),
				},
				{
					WantTimeout:       time.Millisecond * 50,
					WantTimeoutBounds: time.Millisecond * 20,
					WantService:       "serv",
					WantProcedure:     "proc",
					WantBody:          "body",
					GiveRespBody:      "respbody",
				},
			},
			wantBody: "respbody",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			callable := NewOutboundEventCallable(t, tt.events)
			defer callable.Cleanup()

			trans := yarpctest.NewFakeTransport()
			out := trans.NewOutbound(yarpctest.NewFakePeerList(), yarpctest.OutboundCallOverride(callable.Call))
			out.Start()

			retry := NewUnaryMiddleware(
				Retries(tt.retries),
				PerRequestTimeout(tt.retrytimeout),
				BackoffStrategy(tt.retryBackoff),
			)

			ctx := context.Background()
			if tt.reqTimeout != 0 {
				newCtx, cancel := context.WithTimeout(ctx, tt.reqTimeout)
				defer cancel()
				ctx = newCtx
			}
			resp, err := retry.Call(ctx, tt.request, out)
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

func (s *sequentialBackoff) Backoff(attempts uint) time.Duration {
	return time.Duration(s.base.Nanoseconds() * int64(attempts+1))
}
