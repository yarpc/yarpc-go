// Copyright (c) 2019 Uber Technologies, Inc.
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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/digester"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewMiddlewareLogLevels(t *testing.T) {
	// It's a bit unfortunate that we're asserting conditions about the
	// internal state of Middleware and graph here but short of duplicating
	// the other test, this is the cleanest option.

	infoLevel := zapcore.InfoLevel
	warnLevel := zapcore.WarnLevel

	t.Run("Inbound", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			t.Run("default", func(t *testing.T) {
				assert.Equal(t, zapcore.DebugLevel, NewMiddleware(Config{}).graph.inboundLevels.success)
			})

			t.Run("any direction override", func(t *testing.T) {
				assert.Equal(t, zapcore.InfoLevel, NewMiddleware(Config{
					Levels: LevelsConfig{
						Default: DirectionalLevelsConfig{
							Success: &infoLevel,
						},
					},
				}).graph.inboundLevels.success)
			})

			t.Run("directional override", func(t *testing.T) {
				assert.Equal(t, zapcore.InfoLevel, NewMiddleware(Config{
					Levels: LevelsConfig{
						Default: DirectionalLevelsConfig{
							Success: &warnLevel, // overridden by Inbound.Success
						},
						Inbound: DirectionalLevelsConfig{
							Success: &infoLevel, // overrides Default.Success
						},
					},
				}).graph.inboundLevels.success)
			})
		})

		t.Run("Failure", func(t *testing.T) {
			t.Run("default", func(t *testing.T) {
				assert.Equal(t, zapcore.ErrorLevel, NewMiddleware(Config{}).graph.inboundLevels.failure)
			})

			t.Run("override", func(t *testing.T) {
				assert.Equal(t, zapcore.WarnLevel, NewMiddleware(Config{
					Levels: LevelsConfig{
						Inbound: DirectionalLevelsConfig{
							Failure: &warnLevel,
						},
					},
				}).graph.inboundLevels.failure)
			})
		})

		t.Run("ApplicationError", func(t *testing.T) {
			t.Run("default", func(t *testing.T) {
				assert.Equal(t, zapcore.ErrorLevel, NewMiddleware(Config{}).graph.inboundLevels.applicationError)
			})

			t.Run("override", func(t *testing.T) {
				assert.Equal(t, zapcore.WarnLevel, NewMiddleware(Config{
					Levels: LevelsConfig{
						Inbound: DirectionalLevelsConfig{
							ApplicationError: &warnLevel,
						},
					},
				}).graph.inboundLevels.applicationError)
			})
		})
	})

	t.Run("Outbound", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			t.Run("default", func(t *testing.T) {
				assert.Equal(t, zapcore.DebugLevel, NewMiddleware(Config{}).graph.outboundLevels.success)
			})

			t.Run("override", func(t *testing.T) {
				assert.Equal(t, zapcore.InfoLevel, NewMiddleware(Config{
					Levels: LevelsConfig{
						Outbound: DirectionalLevelsConfig{
							Success: &infoLevel,
						},
					},
				}).graph.outboundLevels.success)
			})
		})

		t.Run("Failure", func(t *testing.T) {
			t.Run("default", func(t *testing.T) {
				assert.Equal(t, zapcore.ErrorLevel, NewMiddleware(Config{}).graph.outboundLevels.failure)
			})

			t.Run("override", func(t *testing.T) {
				assert.Equal(t, zapcore.WarnLevel, NewMiddleware(Config{
					Levels: LevelsConfig{
						Outbound: DirectionalLevelsConfig{
							Failure: &warnLevel,
						},
					},
				}).graph.outboundLevels.failure)
			})
		})

		t.Run("ApplicationError", func(t *testing.T) {
			t.Run("default", func(t *testing.T) {
				assert.Equal(t, zapcore.ErrorLevel, NewMiddleware(Config{}).graph.outboundLevels.applicationError)
			})

			t.Run("override", func(t *testing.T) {
				assert.Equal(t, zapcore.WarnLevel, NewMiddleware(Config{
					Levels: LevelsConfig{
						Outbound: DirectionalLevelsConfig{
							ApplicationError: &warnLevel,
						},
					},
				}).graph.outboundLevels.applicationError)
			})
		})
	})
}

func TestMiddlewareLogging(t *testing.T) {
	defer stubTime()()
	req := &transport.Request{
		Caller:          "caller",
		Service:         "service",
		Transport:       "",
		Encoding:        "raw",
		Procedure:       "procedure",
		Headers:         transport.NewHeaders().With("password", "super-secret"),
		ShardKey:        "shard01",
		RoutingKey:      "routing-key",
		RoutingDelegate: "routing-delegate",
		Body:            strings.NewReader("body"),
	}
	sreq := &transport.StreamRequest{Meta: req.ToRequestMeta()}
	failed := errors.New("fail")

	baseFields := func() []zapcore.Field {
		return []zapcore.Field{
			zap.String("source", req.Caller),
			zap.String("dest", req.Service),
			zap.String("transport", unknownIfEmpty(req.Transport)),
			zap.String("procedure", req.Procedure),
			zap.String("encoding", string(req.Encoding)),
			zap.String("routingKey", req.RoutingKey),
			zap.String("routingDelegate", req.RoutingDelegate),
		}
	}

	type test struct {
		desc            string
		err             error // downstream error
		applicationErr  bool  // downstream application error
		wantErrLevel    zapcore.Level
		wantInboundMsg  string
		wantOutboundMsg string
		wantFields      []zapcore.Field
	}

	tests := []test{
		{
			desc:            "no downstream errors",
			wantErrLevel:    zapcore.InfoLevel,
			wantInboundMsg:  "Handled inbound request.",
			wantOutboundMsg: "Made outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", true),
				zap.Skip(),
				zap.Skip(),
			},
		},
		{
			desc:            "downstream transport error",
			err:             failed,
			wantErrLevel:    zapcore.ErrorLevel,
			wantInboundMsg:  "Error handling inbound request.",
			wantOutboundMsg: "Error making outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Skip(),
				zap.Error(failed),
			},
		},
		{
			desc:            "no downstream error but with application error",
			applicationErr:  true,
			wantErrLevel:    zapcore.WarnLevel,
			wantInboundMsg:  "Error handling inbound request.",
			wantOutboundMsg: "Error making outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Skip(),
				zap.String("error", "application_error"),
			},
		},
	}

	newHandler := func(t test) fakeHandler {
		return fakeHandler{err: t.err, applicationErr: t.applicationErr}
	}

	newOutbound := func(t test) fakeOutbound {
		return fakeOutbound{err: t.err, applicationErr: t.applicationErr}
	}

	infoLevel := zapcore.InfoLevel
	warnLevel := zapcore.WarnLevel

	for _, tt := range tests {
		core, logs := observer.New(zapcore.DebugLevel)
		mw := NewMiddleware(Config{
			Logger:           zap.New(core),
			Scope:            metrics.New().Scope(),
			ContextExtractor: NewNopContextExtractor(),
			Levels: LevelsConfig{
				Default: DirectionalLevelsConfig{
					Success:          &infoLevel,
					ApplicationError: &warnLevel,
					// Leave failure level as the default.
				},
			},
		})

		getLog := func(t *testing.T) observer.LoggedEntry {
			entries := logs.TakeAll()
			require.Equal(t, 1, len(entries), "Unexpected number of logs written.")
			e := entries[0]
			e.Entry.Time = time.Time{}
			return e
		}

		checkErr := func(err error) {
			if tt.err != nil {
				assert.Error(t, err, "Expected an error from middleware.")
			} else {
				assert.NoError(t, err, "Unexpected error from middleware.")
			}
		}

		t.Run(tt.desc+", unary inbound", func(t *testing.T) {
			err := mw.Handle(
				context.Background(),
				req,
				&transporttest.FakeResponseWriter{},
				newHandler(tt),
			)
			checkErr(err)
			logContext := append(
				baseFields(),
				zap.String("direction", string(_directionInbound)),
				zap.String("rpcType", "Unary"),
			)
			logContext = append(logContext, tt.wantFields...)
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   tt.wantErrLevel,
					Message: tt.wantInboundMsg,
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(t), "Unexpected log entry written.")
		})
		t.Run(tt.desc+", unary outbound", func(t *testing.T) {
			res, err := mw.Call(context.Background(), req, newOutbound(tt))
			checkErr(err)
			if tt.err == nil {
				assert.NotNil(t, res, "Expected non-nil response if call is successful.")
			}
			logContext := append(
				baseFields(),
				zap.String("direction", string(_directionOutbound)),
				zap.String("rpcType", "Unary"),
			)
			logContext = append(logContext, tt.wantFields...)
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   tt.wantErrLevel,
					Message: tt.wantOutboundMsg,
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(t), "Unexpected log entry written.")
		})

		// Application errors aren't applicable to oneway and streaming
		if tt.applicationErr {
			continue
		}

		t.Run(tt.desc+", oneway inbound", func(t *testing.T) {
			err := mw.HandleOneway(context.Background(), req, newHandler(tt))
			checkErr(err)
			logContext := append(
				baseFields(),
				zap.String("direction", string(_directionInbound)),
				zap.String("rpcType", "Oneway"),
			)
			logContext = append(logContext, tt.wantFields...)
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   tt.wantErrLevel,
					Message: tt.wantInboundMsg,
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(t), "Unexpected log entry written.")
		})
		t.Run(tt.desc+", oneway outbound", func(t *testing.T) {
			ack, err := mw.CallOneway(context.Background(), req, newOutbound(tt))
			checkErr(err)
			logContext := append(
				baseFields(),
				zap.String("direction", string(_directionOutbound)),
				zap.String("rpcType", "Oneway"),
			)
			logContext = append(logContext, tt.wantFields...)
			if tt.err == nil {
				assert.NotNil(t, ack, "Expected non-nil ack if call is successful.")
			}
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   tt.wantErrLevel,
					Message: tt.wantOutboundMsg,
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(t), "Unexpected log entry written.")
		})
		t.Run(tt.desc+", stream inbound", func(t *testing.T) {
			stream, err := transport.NewServerStream(&fakeStream{ctx: context.Background(), request: sreq})
			require.NoError(t, err)
			err = mw.HandleStream(stream, newHandler(tt))
			checkErr(err)
			logContext := append(
				baseFields(),
				zap.String("direction", string(_directionInbound)),
				zap.String("rpcType", "Streaming"),
			)
			logContext = append(logContext, tt.wantFields...)
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   tt.wantErrLevel,
					Message: tt.wantInboundMsg,
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(t), "Unexpected log entry written.")
		})
		t.Run(tt.desc+", stream outbound", func(t *testing.T) {
			clientStream, err := mw.CallStream(context.Background(), sreq, newOutbound(tt))
			checkErr(err)
			logContext := append(
				baseFields(),
				zap.String("direction", string(_directionOutbound)),
				zap.String("rpcType", "Streaming"),
			)
			logContext = append(logContext, tt.wantFields...)
			if tt.err == nil {
				assert.NotNil(t, clientStream, "Expected non-nil clientStream if call is successful.")
			}
			expected := observer.LoggedEntry{
				Entry: zapcore.Entry{
					Level:   tt.wantErrLevel,
					Message: tt.wantOutboundMsg,
				},
				Context: logContext,
			}
			assert.Equal(t, expected, getLog(t), "Unexpected log entry written.")
		})
	}
}

func TestMiddlewareMetrics(t *testing.T) {
	defer stubTime()()
	req := &transport.Request{
		Caller:    "caller",
		Service:   "service",
		Transport: "",
		Encoding:  "raw",
		Procedure: "procedure",
		Body:      strings.NewReader("body"),
	}

	type test struct {
		desc               string
		err                error // downstream error
		applicationErr     bool  // downstream application error
		wantCalls          int
		wantSuccesses      int
		wantCallerFailures map[string]int
		wantServerFailures map[string]int
	}

	tests := []test{
		{
			desc:          "no downstream errors",
			wantCalls:     1,
			wantSuccesses: 1,
		},
		{
			desc:          "invalid argument error",
			err:           yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantCallerFailures: map[string]int{
				yarpcerrors.CodeInvalidArgument.String(): 1,
			},
		},
		{
			desc:          "invalid argument error",
			err:           yarpcerrors.Newf(yarpcerrors.CodeInternal, "test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantServerFailures: map[string]int{
				yarpcerrors.CodeInternal.String(): 1,
			},
		},
		{
			desc:          "unknown (unwrapped) error",
			err:           errors.New("test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantServerFailures: map[string]int{
				"unknown_internal_yarpc": 1,
			},
		},
		{
			desc:          "custom error code error",
			err:           yarpcerrors.Newf(yarpcerrors.Code(1000), "test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantServerFailures: map[string]int{
				"1000": 1,
			},
		},
	}

	newHandler := func(t test) fakeHandler {
		return fakeHandler{err: t.err, applicationErr: t.applicationErr}
	}

	newOutbound := func(t test) fakeOutbound {
		return fakeOutbound{err: t.err, applicationErr: t.applicationErr}
	}

	for _, tt := range tests {
		validate := func(mw *Middleware, direction string, rpcType transport.Type) {
			key, free := getKey(req, direction, rpcType)
			edge := mw.graph.getEdge(key)
			free()
			assert.Equal(t, int64(tt.wantCalls), edge.calls.Load())
			assert.Equal(t, int64(tt.wantSuccesses), edge.successes.Load())
			for tagName, val := range tt.wantCallerFailures {
				assert.Equal(t, int64(val), edge.callerFailures.MustGet(_error, tagName).Load())
			}
			for tagName, val := range tt.wantServerFailures {
				assert.Equal(t, int64(val), edge.serverFailures.MustGet(_error, tagName).Load())
			}
		}
		t.Run(tt.desc+", unary inbound", func(t *testing.T) {
			mw := NewMiddleware(Config{
				Logger:           zap.NewNop(),
				Scope:            metrics.New().Scope(),
				ContextExtractor: NewNopContextExtractor(),
			})
			mw.Handle(
				context.Background(),
				req,
				&transporttest.FakeResponseWriter{},
				newHandler(tt),
			)
			validate(mw, string(_directionInbound), transport.Unary)
		})
		t.Run(tt.desc+", unary outbound", func(t *testing.T) {
			mw := NewMiddleware(Config{
				Logger:           zap.NewNop(),
				Scope:            metrics.New().Scope(),
				ContextExtractor: NewNopContextExtractor(),
			})
			mw.Call(context.Background(), req, newOutbound(tt))
			validate(mw, string(_directionOutbound), transport.Unary)
		})
	}
}

// getKey gets the "key" that we will use to get an edge in the graph.  We use
// a separate function to recreate the logic because extracting it out in the
// main code could have performance implications.
func getKey(req *transport.Request, direction string, rpcType transport.Type) (key []byte, free func()) {
	d := digester.New()
	d.Add(req.Caller)
	d.Add(req.Service)
	d.Add(req.Transport)
	d.Add(string(req.Encoding))
	d.Add(req.Procedure)
	d.Add(req.RoutingKey)
	d.Add(req.RoutingDelegate)
	d.Add(direction)
	d.Add(rpcType.String())
	return d.Digest(), d.Free
}

func TestUnaryInboundApplicationErrors(t *testing.T) {
	defer stubTime()()
	req := &transport.Request{
		Caller:          "caller",
		Service:         "service",
		Transport:       "",
		Encoding:        "raw",
		Procedure:       "procedure",
		ShardKey:        "shard01",
		RoutingKey:      "routing-key",
		RoutingDelegate: "routing-delegate",
		Body:            strings.NewReader("body"),
	}

	expectedFields := []zapcore.Field{
		zap.String("source", req.Caller),
		zap.String("dest", req.Service),
		zap.String("transport", "unknown"),
		zap.String("procedure", req.Procedure),
		zap.String("encoding", string(req.Encoding)),
		zap.String("routingKey", req.RoutingKey),
		zap.String("routingDelegate", req.RoutingDelegate),
		zap.String("direction", string(_directionInbound)),
		zap.String("rpcType", "Unary"),
		zap.Duration("latency", 0),
		zap.Bool("successful", false),
		zap.Skip(),
		zap.String("error", "application_error"),
	}

	core, logs := observer.New(zap.DebugLevel)
	mw := NewMiddleware(Config{
		Logger:           zap.New(core),
		Scope:            metrics.New().Scope(),
		ContextExtractor: NewNopContextExtractor(),
	})

	assert.NoError(t, mw.Handle(
		context.Background(),
		req,
		&transporttest.FakeResponseWriter{},
		fakeHandler{err: nil, applicationErr: true},
	), "Unexpected transport error.")

	expected := observer.LoggedEntry{
		Entry: zapcore.Entry{
			Level:   zapcore.ErrorLevel,
			Message: "Error handling inbound request.",
		},
		Context: expectedFields,
	}
	entries := logs.TakeAll()
	require.Equal(t, 1, len(entries), "Unexpected number of log entries written.")
	entry := entries[0]
	entry.Time = time.Time{}
	assert.Equal(t, expected, entry, "Unexpected log entry written.")
}

func TestMiddlewareSuccessSnapshot(t *testing.T) {
	defer stubTime()()
	root := metrics.New()
	meter := root.Scope()
	mw := NewMiddleware(Config{
		Logger:           zap.NewNop(),
		Scope:            meter,
		ContextExtractor: NewNopContextExtractor(),
	})

	err := mw.Handle(
		context.Background(),
		&transport.Request{
			Caller:          "caller",
			Service:         "service",
			Transport:       "",
			Encoding:        "raw",
			Procedure:       "procedure",
			ShardKey:        "sk",
			RoutingKey:      "rk",
			RoutingDelegate: "rd",
			Body:            strings.NewReader("body"),
		},
		&transporttest.FakeResponseWriter{},
		fakeHandler{nil, false},
	)
	assert.NoError(t, err, "Unexpected transport error.")

	snap := root.Snapshot()
	tags := metrics.Tags{
		"dest":             "service",
		"direction":        "inbound",
		"transport":        "unknown",
		"encoding":         "raw",
		"procedure":        "procedure",
		"routing_delegate": "rd",
		"routing_key":      "rk",
		"rpc_type":         transport.Unary.String(),
		"source":           "caller",
	}
	want := &metrics.RootSnapshot{
		Counters: []metrics.Snapshot{
			{Name: "calls", Tags: tags, Value: 1},
			{Name: "successes", Tags: tags, Value: 1},
		},
		Histograms: []metrics.HistogramSnapshot{
			{
				Name: "caller_failure_latency_ms",
				Tags: tags,
				Unit: time.Millisecond,
			},
			{
				Name: "server_failure_latency_ms",
				Tags: tags,
				Unit: time.Millisecond,
			},
			{
				Name:   "success_latency_ms",
				Tags:   tags,
				Unit:   time.Millisecond,
				Values: []int64{1},
			},
		},
	}
	assert.Equal(t, want, snap, "Unexpected snapshot of metrics.")
}

func TestMiddlewareFailureSnapshot(t *testing.T) {
	defer stubTime()()
	root := metrics.New()
	meter := root.Scope()
	mw := NewMiddleware(Config{
		Logger:           zap.NewNop(),
		Scope:            meter,
		ContextExtractor: NewNopContextExtractor(),
	})

	err := mw.Handle(
		context.Background(),
		&transport.Request{
			Caller:          "caller",
			Service:         "service",
			Transport:       "",
			Encoding:        "raw",
			Procedure:       "procedure",
			ShardKey:        "sk",
			RoutingKey:      "rk",
			RoutingDelegate: "rd",
			Body:            strings.NewReader("body"),
		},
		&transporttest.FakeResponseWriter{},
		fakeHandler{fmt.Errorf("yuno"), false},
	)
	assert.Error(t, err, "Expected transport error.")

	snap := root.Snapshot()
	tags := metrics.Tags{
		"dest":             "service",
		"direction":        "inbound",
		"encoding":         "raw",
		"procedure":        "procedure",
		"routing_delegate": "rd",
		"routing_key":      "rk",
		"rpc_type":         transport.Unary.String(),
		"source":           "caller",
		"transport":        "unknown",
	}
	errorTags := metrics.Tags{
		"dest":             "service",
		"direction":        "inbound",
		"encoding":         "raw",
		"error":            "unknown_internal_yarpc",
		"procedure":        "procedure",
		"routing_delegate": "rd",
		"routing_key":      "rk",
		"rpc_type":         transport.Unary.String(),
		"source":           "caller",
		"transport":        "unknown",
	}
	want := &metrics.RootSnapshot{
		Counters: []metrics.Snapshot{
			{Name: "calls", Tags: tags, Value: 1},
			{Name: "server_failures", Tags: errorTags, Value: 1},
			{Name: "successes", Tags: tags, Value: 0},
		},
		Histograms: []metrics.HistogramSnapshot{
			{
				Name: "caller_failure_latency_ms",
				Tags: tags,
				Unit: time.Millisecond,
			},
			{
				Name:   "server_failure_latency_ms",
				Tags:   tags,
				Unit:   time.Millisecond,
				Values: []int64{1},
			},
			{
				Name: "success_latency_ms",
				Tags: tags,
				Unit: time.Millisecond,
			},
		},
	}
	assert.Equal(t, want, snap, "Unexpected snapshot of metrics.")
}

func TestApplicationErrorSnapShot(t *testing.T) {
	defer stubTime()()

	tests := []struct {
		name   string
		err    error
		errTag string
		appErr bool
	}{
		{
			name:   "status", // eg error returned in transport middleware
			err:    yarpcerrors.Newf(yarpcerrors.CodeAlreadyExists, "foo exists!"),
			errTag: "already-exists",
		},
		{
			name:   "status and app error", // eg Protobuf handler returning yarpcerrors.Status
			err:    yarpcerrors.Newf(yarpcerrors.CodeAlreadyExists, "foo exists!"),
			errTag: "already-exists",
			appErr: true,
		},
		{
			name:   "no status and app error", // eg Thrift exception
			err:    errors.New("foo-bar-baz"),
			errTag: "application_error",
			appErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := metrics.New()
			meter := root.Scope()
			mw := NewMiddleware(Config{
				Logger: zap.NewNop(),
				Scope:  meter,
			})

			err := mw.Handle(
				context.Background(),
				&transport.Request{
					Caller:          "caller",
					Service:         "service",
					Transport:       "",
					Encoding:        "raw",
					Procedure:       "procedure",
					ShardKey:        "sk",
					RoutingKey:      "rk",
					RoutingDelegate: "rd",
				},
				&transporttest.FakeResponseWriter{},
				fakeHandler{
					err:            tt.err,
					applicationErr: tt.appErr,
				},
			)
			require.Error(t, err)

			snap := root.Snapshot()
			tags := metrics.Tags{
				"dest":             "service",
				"direction":        "inbound",
				"transport":        "unknown",
				"encoding":         "raw",
				"procedure":        "procedure",
				"routing_delegate": "rd",
				"routing_key":      "rk",
				"rpc_type":         transport.Unary.String(),
				"source":           "caller",
			}
			errorTags := metrics.Tags{
				"dest":             "service",
				"direction":        "inbound",
				"transport":        "unknown",
				"encoding":         "raw",
				"procedure":        "procedure",
				"routing_delegate": "rd",
				"routing_key":      "rk",
				"rpc_type":         transport.Unary.String(),
				"source":           "caller",
				"error":            tt.errTag,
			}
			want := &metrics.RootSnapshot{
				Counters: []metrics.Snapshot{
					{Name: "caller_failures", Tags: errorTags, Value: 1},
					{Name: "calls", Tags: tags, Value: 1},
					{Name: "successes", Tags: tags, Value: 0},
				},
				Histograms: []metrics.HistogramSnapshot{
					{
						Name:   "caller_failure_latency_ms",
						Tags:   tags,
						Unit:   time.Millisecond,
						Values: []int64{1},
					},
					{
						Name: "server_failure_latency_ms",
						Tags: tags,
						Unit: time.Millisecond,
					},
					{
						Name: "success_latency_ms",
						Tags: tags,
						Unit: time.Millisecond,
					},
				},
			}
			assert.Equal(t, want, snap, "Unexpected snapshot of metrics.")

		})
	}
}
