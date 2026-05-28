// Copyright (c) 2026 Uber Technologies, Inc.
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

package transport_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
)

func TestInvokeUnaryHandlerContextErrors(t *testing.T) {
	req := &transport.Request{
		Caller:    "caller",
		Service:   "service",
		Procedure: "procedure",
	}

	startTime := time.Now().Add(-2 * time.Second)
	expiredCtx, cancel := context.WithDeadline(context.Background(), startTime.Add(time.Second))
	defer cancel()

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name              string
		ctx               context.Context
		startTime         time.Time
		handlerErr        error
		expectedErr       error
		expectedYARPCCode *yarpcerrors.Code
		expectedMessage   string
	}{
		{
			name:       "context canceled maps to yarpc cancelled",
			ctx:        canceledCtx,
			startTime:  time.Now(),
			handlerErr: context.Canceled,
			expectedYARPCCode: func() *yarpcerrors.Code {
				code := yarpcerrors.CodeCancelled
				return &code
			}(),
			expectedMessage: `call to procedure "procedure" of service "service" from caller "caller" was cancelled`,
		},
		{
			name:        "context canceled passes through when context is not canceled",
			ctx:         context.Background(),
			startTime:   time.Now(),
			handlerErr:  context.Canceled,
			expectedErr: context.Canceled,
		},
		{
			name:       "deadline exceeded maps to yarpc deadline exceeded",
			ctx:        expiredCtx,
			startTime:  startTime,
			handlerErr: context.DeadlineExceeded,
			expectedYARPCCode: func() *yarpcerrors.Code {
				code := yarpcerrors.CodeDeadlineExceeded
				return &code
			}(),
			expectedMessage: `call to procedure "procedure" of service "service" from caller "caller" timed out after 1s`,
		},
		{
			name:        "deadline exceeded passes through when context is not expired",
			ctx:         context.Background(),
			startTime:   time.Now(),
			handlerErr:  context.DeadlineExceeded,
			expectedErr: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := transport.InvokeUnaryHandler(transport.UnaryInvokeRequest{
				Context:   tt.ctx,
				StartTime: tt.startTime,
				Request:   req,
				Handler: transport.UnaryHandlerFunc(func(context.Context, *transport.Request, transport.ResponseWriter) error {
					return tt.handlerErr
				}),
				Logger: zap.NewNop(),
			})
			require.Error(t, err)

			if tt.expectedYARPCCode != nil {
				yarpcErr := yarpcerrors.FromError(err)
				require.NotNil(t, yarpcErr)
				assert.Equal(t, *tt.expectedYARPCCode, yarpcErr.Code())
				assert.Equal(t, tt.expectedMessage, yarpcErr.Message())
				return
			}

			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
