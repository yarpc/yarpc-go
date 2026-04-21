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

package yarpc

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func loggedStringFields(e observer.LoggedEntry) map[string]string {
	m := make(map[string]string)
	for _, f := range e.Context {
		if f.Type == zapcore.StringType {
			m[f.Key] = f.String
		}
	}
	return m
}

func TestRegister_thriftUnannotatedExceptionWarnLogs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	core, logs := observer.New(zapcore.WarnLevel)
	const procName = "mysvc::ping"
	d := NewDispatcher(Config{
		Name:    "mysvc",
		Logging: LoggingConfig{Zap: zap.New(core)},
	})
	require.Nil(t, d.meter)

	h := transporttest.NewMockUnaryHandler(mockCtrl)
	assert.NotPanics(t, func() {
		d.Register([]transport.Procedure{{
			Name:        procName,
			Encoding:    transport.ThriftEncoding,
			HandlerSpec: transport.NewUnaryHandlerSpec(h),
			Exceptions: map[string]string{
				"error.InvalidArgumentError": transport.RPCCodeNotSetLiteral,
			},
		}})
	})

	w := logs.FilterMessage("Registered procedure may throw an unannotated Thrift exception.")
	require.Equal(t, 1, w.Len())
	e := w.All()[0]
	fields := loggedStringFields(e)
	assert.Equal(t, procName, fields["procedure"])
	assert.Equal(t, "error.InvalidArgumentError", fields["exception"])
}

func TestRegister_thriftUnannotatedExceptionWithMetricsScope(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	core, logs := observer.New(zapcore.WarnLevel)
	const procName = "mysvc::ping"
	d := NewDispatcher(Config{
		Name: "mysvc",
		Metrics: MetricsConfig{
			Metrics: metrics.New().Scope(),
		},
		Logging: LoggingConfig{Zap: zap.New(core)},
	})
	require.NotNil(t, d.meter)

	h := transporttest.NewMockUnaryHandler(mockCtrl)
	assert.NotPanics(t, func() {
		d.Register([]transport.Procedure{{
			Name:        procName,
			Encoding:    transport.ThriftEncoding,
			HandlerSpec: transport.NewUnaryHandlerSpec(h),
			Exceptions: map[string]string{
				"error.InvalidArgumentError": transport.RPCCodeNotSetLiteral,
			},
		}})
	})

	w := logs.FilterMessage("Registered procedure may throw an unannotated Thrift exception.")
	require.Equal(t, 1, w.Len())
	e := w.All()[0]
	fields := loggedStringFields(e)
	assert.Equal(t, procName, fields["procedure"])
	assert.Equal(t, "error.InvalidArgumentError", fields["exception"])
}

func TestRegister_thriftRpcCodeAnnotationSkipsUnannotatedWarn(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	core, logs := observer.New(zapcore.WarnLevel)
	d := NewDispatcher(Config{
		Name:    "mysvc",
		Logging: LoggingConfig{Zap: zap.New(core)},
	})
	h := transporttest.NewMockUnaryHandler(mockCtrl)
	d.Register([]transport.Procedure{{
		Name:        "mysvc::ping",
		Encoding:    transport.ThriftEncoding,
		HandlerSpec: transport.NewUnaryHandlerSpec(h),
		Exceptions: map[string]string{
			"SomeErr": "INVALID_ARGUMENT",
		},
	}})
	assert.Equal(t, 0, logs.Len())
}

func TestRegister_nonThriftEncodingSkipsExceptionInstrumentation(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	core, logs := observer.New(zapcore.WarnLevel)
	d := NewDispatcher(Config{
		Name:    "mysvc",
		Logging: LoggingConfig{Zap: zap.New(core)},
	})
	h := transporttest.NewMockUnaryHandler(mockCtrl)
	d.Register([]transport.Procedure{{
		Name:        "mysvc::ping",
		Encoding:    "json",
		HandlerSpec: transport.NewUnaryHandlerSpec(h),
		Exceptions: map[string]string{
			"SomeErr": transport.RPCCodeNotSetLiteral,
		},
	}})
	assert.Equal(t, 0, logs.Len())
}

func TestRegister_thriftNilExceptionsNoWarn(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	core, logs := observer.New(zapcore.WarnLevel)
	d := NewDispatcher(Config{
		Name:    "mysvc",
		Logging: LoggingConfig{Zap: zap.New(core)},
	})
	h := transporttest.NewMockUnaryHandler(mockCtrl)
	d.Register([]transport.Procedure{{
		Name:        "mysvc::ping",
		Encoding:    transport.ThriftEncoding,
		HandlerSpec: transport.NewUnaryHandlerSpec(h),
	}})
	assert.Equal(t, 0, logs.Len())
}

func TestRegister_thriftMultipleUnannotatedExceptionsWarnEach(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	core, logs := observer.New(zapcore.WarnLevel)
	d := NewDispatcher(Config{
		Name:    "mysvc",
		Logging: LoggingConfig{Zap: zap.New(core)},
	})
	h := transporttest.NewMockUnaryHandler(mockCtrl)
	d.Register([]transport.Procedure{{
		Name:        "mysvc::ping",
		Encoding:    transport.ThriftEncoding,
		HandlerSpec: transport.NewUnaryHandlerSpec(h),
		Exceptions: map[string]string{
			"AErr": transport.RPCCodeNotSetLiteral,
			"BErr": transport.RPCCodeNotSetLiteral,
		},
	}})
	w := logs.FilterMessage("Registered procedure may throw an unannotated Thrift exception.")
	require.Equal(t, 2, w.Len())
	got := make(map[string]struct{})
	for _, e := range w.All() {
		got[loggedStringFields(e)["exception"]] = struct{}{}
	}
	assert.Contains(t, got, "AErr")
	assert.Contains(t, got, "BErr")
}
