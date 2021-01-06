// Copyright (c) 2021 Uber Technologies, Inc.
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
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestContextMiddleware(t *testing.T) {
	const (
		ctxDeadlineExceededMsg = `call to procedure "my-procedure" of service "my-service" from caller "my-caller" timed out`
		ctxCancelledMsg        = `call to procedure "my-procedure" of service "my-service" from caller "my-caller" was canceled`
	)

	core, logs := observer.New(zapcore.DebugLevel)
	infoLevel := zapcore.InfoLevel
	mw := NewMiddleware(Config{
		Logger:           zap.New(core),
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
		appErr     bool
		ctx        func() context.Context

		wantDeadlineExceeded bool
		wantCtxCancelled     bool
	}{
		{
			name: "no-op/handler success",
			ctx:  func() context.Context { return context.Background() },
		},
		{
			name:       "no-op/handler err",
			handlerErr: errors.New("an err"),
			ctx:        func() context.Context { return context.Background() },
		},
		{
			name: "deadline exceeded/handler success",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), -1)
				cancel()
				return ctx
			},
			wantDeadlineExceeded: true,
		},
		{
			name:       "deadline exceeded/handler err",
			handlerErr: fmt.Errorf("my custom error"),
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), -1)
				cancel()
				return ctx
			},
			wantDeadlineExceeded: true,
		},
		{
			name: "deadline exceeded/app err",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), -1)
				cancel()
				return ctx
			},
			appErr:               true,
			wantDeadlineExceeded: true,
		},
		{
			name: "cancelled error/handler success",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantCtxCancelled: true,
		},
		{
			name:       "cancelled error/handler err",
			handlerErr: fmt.Errorf("my custom error"),
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantCtxCancelled: true,
		},
		{
			name: "cancelled error/app err",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			appErr:           true,
			wantCtxCancelled: true,
		},
	}

	req := &transport.Request{
		Service:   "my-service",
		Procedure: "my-procedure",
		Caller:    "my-caller",
	}

	expectLogField := func(appErr bool, err error) *zap.Field {
		dropMsg := _droppedSuccessLog
		if err == nil && appErr {
			dropMsg = _droppedAppErrLog
		} else if err != nil {
			dropMsg = fmt.Sprintf(_droppedErrLogFmt, err)
		}
		log := zap.String(_dropped, dropMsg)
		return &log
	}

	getDropLogField := func(t *testing.T) *zap.Field {
		entries := logs.TakeAll()
		require.Equal(t, 1, len(entries), "unexpected number of logs written: %v", entries)
		for _, f := range entries[0].Context {
			if f.Key == _dropped {
				return &f
			}
		}
		return nil
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer logs.TakeAll() // throw away logs for next run

			handler := &testHandler{err: tt.handlerErr, appErr: tt.appErr}
			err := mw.Handle(tt.ctx(), req, &transporttest.FakeResponseWriter{}, handler)

			if tt.wantDeadlineExceeded {
				assert.EqualError(t,
					err,
					yarpcerrors.DeadlineExceededErrorf(ctxDeadlineExceededMsg).Error(),
					"expected deadline exceeded error override")

				assert.Equal(t, expectLogField(tt.appErr, tt.handlerErr), getDropLogField(t), "unexpected log")
				return
			}

			if tt.wantCtxCancelled {
				assert.EqualError(t,
					err,
					yarpcerrors.CancelledErrorf(ctxCancelledMsg).Error(),
					"expected cancelled yarpcerror code")

				assert.Equal(t, expectLogField(tt.appErr, tt.handlerErr), getDropLogField(t), "unexpected log")
				return
			}

			assert.Equal(t, tt.handlerErr, err, "unexpected error")
			assert.Nil(t, getDropLogField(t), "unexpectedly saw 'dropped' log field")
		})
	}
}

type testHandler struct {
	err    error
	appErr bool
}

func (h *testHandler) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	if h.appErr {
		resw.SetApplicationError()
	}
	return h.err
}
