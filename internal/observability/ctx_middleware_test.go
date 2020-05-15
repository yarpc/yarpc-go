// Copyright (c) 2020 Uber Technologies, Inc.
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

package observability

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestContextMiddleware(t *testing.T) {
	const (
		ctxDeadlineExceededMsg = `call to procedure "my-procedure" of service "my-service" from caller "my-caller" timed out`
		ctxCancelledMsg        = `call to procedure "my-procedure" of service "my-service" from caller "my-caller" was canceled`
	)

	infoLevel := zapcore.InfoLevel

	mw := NewMiddleware(Config{
		Logger:           zap.NewNop(),
		ContextExtractor: NewNopContextExtractor(),
		Levels: LevelsConfig{
			Default: DirectionalLevelsConfig{
				Success:          &infoLevel,
				ApplicationError: &infoLevel,
				Failure:          &infoLevel,
			},
		},
	})

	tests := []struct {
		name       string
		handlerErr error
		ctx        func() context.Context

		wantDeadlineExceeded bool
		wantCtxCancelled     bool
	}{
		{
			name: "no-op with no handler err",
			ctx:  func() context.Context { return context.Background() },
		},
		{
			name:       "no-op with handler err",
			handlerErr: errors.New("an err"),
			ctx:        func() context.Context { return context.Background() },
		},
		{
			name:       "ctx deadline exceeded error",
			handlerErr: fmt.Errorf("my custom error"),
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), -1)
				cancel()
				return ctx
			},
			wantDeadlineExceeded: true,
		},
		{
			name:       "ctx cancelled error",
			handlerErr: fmt.Errorf("my custom error"),
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantCtxCancelled: true,
		},
	}

	req := &transport.Request{
		Service:   "my-service",
		Procedure: "my-procedure",
		Caller:    "my-caller",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &testHandler{err: tt.handlerErr}
			err := mw.Handle(tt.ctx(), req, &transporttest.FakeResponseWriter{}, handler)

			if tt.wantDeadlineExceeded {
				assert.EqualError(t,
					err,
					yarpcerrors.DeadlineExceededErrorf(ctxDeadlineExceededMsg).Error(),
					"expected deadline exceeded error override")
				return
			}

			if tt.wantCtxCancelled {
				assert.EqualError(t,
					err,
					yarpcerrors.CancelledErrorf(ctxCancelledMsg).Error(),
					"expected cancelled yarpcerror code")
				return
			}

			assert.Equal(t, tt.handlerErr, err, "unexpected error")
		})
	}
}

type testHandler struct {
	err error
}

func (h *testHandler) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	return h.err
}
