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
	}
}

func TestMiddlewareStreamingLogging(t *testing.T) {
	defer stubTime()()
	req := &transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Caller:          "caller",
			Service:         "service",
			Transport:       "transport",
			Encoding:        "raw",
			Procedure:       "procedure",
			Headers:         transport.NewHeaders().With("hello!", "goodbye!"),
			ShardKey:        "shard-key",
			RoutingKey:      "routing-key",
			RoutingDelegate: "routing-delegate",
		},
	}

	// helper function to creating logging fields for assertion
	newZapFields := func(extraFields ...zapcore.Field) []zapcore.Field {
		fields := []zapcore.Field{
			zap.String("source", req.Meta.Caller),
			zap.String("dest", req.Meta.Service),
			zap.String("transport", req.Meta.Transport),
			zap.String("procedure", req.Meta.Procedure),
			zap.String("encoding", string(req.Meta.Encoding)),
			zap.String("routingKey", req.Meta.RoutingKey),
			zap.String("routingDelegate", req.Meta.RoutingDelegate),
		}
		return append(fields, extraFields...)
	}

	// create middleware
	core, logs := observer.New(zapcore.DebugLevel)
	infoLevel := zapcore.InfoLevel
	mw := NewMiddleware(Config{
		Logger:           zap.New(core),
		Scope:            metrics.New().Scope(),
		ContextExtractor: NewNopContextExtractor(),
		Levels: LevelsConfig{
			Default: DirectionalLevelsConfig{Success: &infoLevel},
		},
	})

	// helper function to retrive observered logs, asserting the expected number
	getLogs := func(t *testing.T, num int) []observer.LoggedEntry {
		logs := logs.TakeAll()
		require.Equal(t, num, len(logs), "expected exactly %d logs, got %v: %#v", num, len(logs), logs)

		var entries []observer.LoggedEntry
		for _, e := range logs {
			// zero the time for easiy comparisons
			e.Entry.Time = time.Time{}
			entries = append(entries, e)
		}
		return entries
	}

	t.Run("success server", func(t *testing.T) {
		stream, err := transport.NewServerStream(&fakeStream{request: req})
		require.NoError(t, err)

		err = mw.HandleStream(stream, &fakeHandler{
			// send and receive messages in the handler
			handleStream: func(stream *transport.ServerStream) {
				err := stream.SendMessage(context.Background(), nil /*message*/)
				require.NoError(t, err)
				_, err = stream.ReceiveMessage(context.Background())
				require.NoError(t, err)
			}})
		require.NoError(t, err)

		logFields := func() []zapcore.Field {
			return newZapFields(
				zap.String("direction", string(_directionInbound)),
				zap.String("rpcType", "Streaming"),
				zap.Bool("successful", true),
				zap.Skip(), // context extractor
				zap.Skip(), // nil error
			)
		}

		wantLogs := []observer.LoggedEntry{
			{
				// open stream
				Entry: zapcore.Entry{
					Message: _successStreamOpen,
				},
				Context: logFields(),
			},
			{
				// send message
				Entry: zapcore.Entry{
					Message: _successfulStreamSend,
				},
				Context: logFields(),
			},
			{
				// receive message
				Entry: zapcore.Entry{
					Message: _successfulStreamReceive,
				},
				Context: logFields(),
			},
			{
				// close stream
				Entry: zapcore.Entry{
					Message: _successStreamClose,
				},
				Context: append(logFields(), zap.Duration("duration", 0)),
			},
		}

		// log 1 - open stream
		// log 2 - send message
		// log 3 - receive message
		// log 4 - close stream
		gotLogs := getLogs(t, 4)
		assert.Equal(t, wantLogs, gotLogs)
	})

	t.Run("error handler", func(t *testing.T) {
		tests := []struct {
			name string
			err  error
		}{
			{
				name: "client fault",
				err:  yarpcerrors.InvalidArgumentErrorf("client err"),
			},
			{
				name: "server fault",
				err:  yarpcerrors.InternalErrorf("server err"),
			},
			{
				name: "unknown fault",
				err:  errors.New("unknown fault"),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				stream, err := transport.NewServerStream(&fakeStream{request: req})
				require.NoError(t, err)

				err = mw.HandleStream(stream, &fakeHandler{err: tt.err})
				require.Error(t, err)

				fields := newZapFields(
					zap.String("direction", string(_directionInbound)),
					zap.String("rpcType", "Streaming"),
					zap.Bool("successful", false),
					zap.Skip(), // context extractor
					zap.Error(tt.err),
					zap.Duration("duration", 0),
				)

				wantLog := observer.LoggedEntry{
					Entry: zapcore.Entry{
						Message: _errorStreamClose,
						Level:   zapcore.ErrorLevel,
					},
					Context: fields,
				}

				// The stream handler is only executed after a stream successfully connects
				// with a client. Therefore the first streaming log will always be
				// successful (tested in the previous subtest). We only care about the
				// stream termination so we retrieve the last log.
				//
				// log 1 - open stream
				// log 2 - close stream
				gotLog := getLogs(t, 2)[1]
				assert.Equal(t, wantLog, gotLog)
			})
		}
	})

	t.Run("error server - send and recv", func(t *testing.T) {
		sendErr := errors.New("send err")
		receiveErr := errors.New("receive err")

		stream, err := transport.NewServerStream(&fakeStream{
			request:    req,
			sendErr:    sendErr,
			receiveErr: receiveErr,
		})
		require.NoError(t, err)

		err = mw.HandleStream(stream, &fakeHandler{
			// send and receive messages in the handler
			handleStream: func(stream *transport.ServerStream) {
				err := stream.SendMessage(context.Background(), nil /*message*/)
				require.Error(t, err)
				_, err = stream.ReceiveMessage(context.Background())
				require.Error(t, err)
			}})
		require.NoError(t, err)

		fields := func() []zapcore.Field {
			return newZapFields(
				zap.String("direction", string(_directionInbound)),
				zap.String("rpcType", "Streaming"),
				zap.Bool("successful", false),
				zap.Skip(), // context extractor
			)
		}

		wantLogs := []observer.LoggedEntry{
			{
				// send message
				Entry: zapcore.Entry{
					Message: _errorStreamSend,
					Level:   zapcore.ErrorLevel,
				},
				Context: append(fields(), zap.Error(sendErr)),
			},
			{
				// receive message
				Entry: zapcore.Entry{
					Message: _errorStreamReceive,
					Level:   zapcore.ErrorLevel,
				},
				Context: append(fields(), zap.Error(receiveErr)),
			},
		}

		// We are only interested in the send and receive logs.
		// log 1 - open stream
		// log 2 - send message
		// log 3 - receive message
		// log 4 - close stream
		gotLogs := getLogs(t, 4)[1:3]
		assert.Equal(t, wantLogs, gotLogs)
	})

	t.Run("success client", func(t *testing.T) {
		stream, err := mw.CallStream(context.Background(), req, fakeOutbound{})
		require.NoError(t, err)
		err = stream.SendMessage(context.Background(), nil /* message */)
		require.NoError(t, err)
		_, err = stream.ReceiveMessage(context.Background())
		require.NoError(t, err)
		require.NoError(t, stream.Close(context.Background()))

		fields := func() []zapcore.Field {
			return newZapFields(
				zap.String("direction", string(_directionOutbound)),
				zap.String("rpcType", "Streaming"),
				zap.Bool("successful", true),
				zap.Skip(), // context extractor
				zap.Skip(), // nil error
			)
		}

		wantLogs := []observer.LoggedEntry{
			{
				// stream open
				Entry: zapcore.Entry{
					Message: _successStreamOpen,
				},
				Context: fields(),
			},
			{
				// stream send
				Entry: zapcore.Entry{
					Message: _successfulStreamSend,
				},
				Context: fields(),
			},
			{
				// stream receive
				Entry: zapcore.Entry{
					Message: _successfulStreamReceive,
				},
				Context: fields(),
			},
			{
				// stream close
				Entry: zapcore.Entry{
					Message: _successStreamClose,
				},
				Context: append(fields(), zap.Duration("duration", 0)),
			},
		}

		// log 1 - open stream
		// log 2 - send message
		// log 3 - receive message
		// log 4 - close stream
		gotLogs := getLogs(t, 4)
		assert.Equal(t, wantLogs, gotLogs)
	})

	t.Run("error client handshake", func(t *testing.T) {
		clientErr := errors.New("client err")
		_, err := mw.CallStream(context.Background(), req, fakeOutbound{err: clientErr})
		require.Error(t, err)

		fields := func() []zapcore.Field {
			return newZapFields(
				zap.String("direction", string(_directionOutbound)),
				zap.String("rpcType", "Streaming"),
				zap.Bool("successful", false),
				zap.Skip(), // context extractor
				zap.Error(clientErr),
			)
		}

		wantLogs := []observer.LoggedEntry{
			{
				// stream open
				Entry: zapcore.Entry{
					Message: _errorStreamOpen,
					Level:   zapcore.ErrorLevel,
				},
				Context: fields(),
			},
		}

		// log 1 - open stream
		gotLogs := getLogs(t, 1)
		assert.Equal(t, wantLogs, gotLogs)
	})

	t.Run("error client - send recv close", func(t *testing.T) {
		sendErr := errors.New("send err")
		receiveErr := errors.New("receive err")
		closeErr := errors.New("close err")

		stream, err := mw.CallStream(context.Background(), req, fakeOutbound{
			stream: fakeStream{
				sendErr:    sendErr,
				receiveErr: receiveErr,
				closeErr:   closeErr,
			}})
		require.NoError(t, err)

		err = stream.SendMessage(context.Background(), nil /* message */)
		require.Error(t, err)
		_, err = stream.ReceiveMessage(context.Background())
		require.Error(t, err)
		err = stream.Close(context.Background())
		require.Error(t, err)

		fields := func() []zapcore.Field {
			return newZapFields(
				zap.String("direction", string(_directionOutbound)),
				zap.String("rpcType", "Streaming"),
				zap.Bool("successful", false),
				zap.Skip(), // context extractor
			)
		}

		wantLogs := []observer.LoggedEntry{
			{
				// send message
				Entry: zapcore.Entry{
					Message: _errorStreamSend,
					Level:   zapcore.ErrorLevel,
				},
				Context: append(fields(), zap.Error(sendErr)),
			},
			{
				// receive message
				Entry: zapcore.Entry{
					Message: _errorStreamReceive,
					Level:   zapcore.ErrorLevel,
				},
				Context: append(fields(), zap.Error(receiveErr)),
			},
			{
				// close stream
				Entry: zapcore.Entry{
					Message: _errorStreamClose,
					Level:   zapcore.ErrorLevel,
				},
				Context: append(fields(), zap.Error(closeErr), zap.Duration("duration", 0)),
			},
		}

		// We are only interested in the send, receive and stream close logs
		// log 1 - open stream
		// log 2 - send message
		// log 3 - receive message
		// log 4 - close stream
		gotLogs := getLogs(t, 4)[1:]
		assert.Equal(t, wantLogs, gotLogs)
	})
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
			assert.Equal(t, int64(0), edge.panics.Load())
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
		fakeHandler{err: nil, applicationErr: false},
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
			{Name: "panics", Tags: tags, Value: 0},
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
		fakeHandler{err: fmt.Errorf("yuno"), applicationErr: false},
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
			{Name: "panics", Tags: tags, Value: 0},
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
					{Name: "panics", Tags: tags, Value: 0},
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

func TestStreamingMetrics(t *testing.T) {
	defer stubTime()()

	req := &transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Caller:          "caller",
			Service:         "service",
			Transport:       "",
			Encoding:        "raw",
			Procedure:       "procedure",
			ShardKey:        "sk",
			RoutingKey:      "rk",
			RoutingDelegate: "rd",
		},
	}

	newTags := func(direction directionName, withErr string) metrics.Tags {
		tags := metrics.Tags{
			"dest":             "service",
			"direction":        string(direction),
			"encoding":         "raw",
			"procedure":        "procedure",
			"routing_delegate": "rd",
			"routing_key":      "rk",
			"rpc_type":         transport.Streaming.String(),
			"source":           "caller",
			"transport":        "unknown",
		}
		if withErr != "" {
			tags["error"] = withErr
		}
		return tags
	}

	t.Run("success server", func(t *testing.T) {
		root := metrics.New()
		scope := root.Scope()
		mw := NewMiddleware(Config{
			Logger:           zap.NewNop(),
			Scope:            scope,
			ContextExtractor: NewNopContextExtractor(),
		})

		stream, err := transport.NewServerStream(&fakeStream{request: req})
		require.NoError(t, err)
		err = mw.HandleStream(stream, &fakeHandler{
			handleStream: func(stream *transport.ServerStream) {
				err := stream.SendMessage(context.Background(), nil /*message*/)
				require.NoError(t, err)
				_, err = stream.ReceiveMessage(context.Background())
				require.NoError(t, err)
			}})
		require.NoError(t, err)

		snap := root.Snapshot()
		tags := newTags(_directionInbound, "")

		// successful handshake, send, recv and close
		want := &metrics.RootSnapshot{
			Counters: []metrics.Snapshot{
				{Name: "calls", Tags: tags, Value: 1},
				{Name: "panics", Tags: tags, Value: 0},
				{Name: "stream_receive_successes", Tags: tags, Value: 1},
				{Name: "stream_receives", Tags: tags, Value: 1},
				{Name: "stream_send_successes", Tags: tags, Value: 1},
				{Name: "stream_sends", Tags: tags, Value: 1},
				{Name: "successes", Tags: tags, Value: 1},
			},
			Gauges: []metrics.Snapshot{
				{Name: "streams_active", Tags: tags, Value: 0}, // opened (+1) then closed (-1)
			},
			Histograms: []metrics.HistogramSnapshot{
				{Name: "stream_duration_ms", Tags: tags, Unit: time.Millisecond, Values: []int64{1}},
			},
		}
		assert.Equal(t, want, snap, "unexpected metrics snapshot")
	})

	t.Run("error handler", func(t *testing.T) {
		tests := []struct {
			name    string
			panics  bool
			err     error
			errName string
		}{
			{
				name:    "client fault",
				err:     yarpcerrors.InvalidArgumentErrorf("client err"),
				errName: yarpcerrors.CodeInvalidArgument.String(),
			},
			{
				name:    "server fault",
				err:     yarpcerrors.InternalErrorf("server err"),
				errName: yarpcerrors.CodeInternal.String(),
			},
			{
				name:    "unknown fault",
				err:     errors.New("unknown fault"),
				errName: "unknown_internal_yarpc",
			},
			{
				name:   "server panic",
				panics: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				root := metrics.New()
				scope := root.Scope()
				mw := NewMiddleware(Config{
					Logger:           zap.NewNop(),
					Scope:            scope,
					ContextExtractor: NewNopContextExtractor(),
				})

				stream, err := transport.NewServerStream(&fakeStream{request: req})
				require.NoError(t, err)

				handleStreamAndValidateFn := func() {
					err = mw.HandleStream(stream, &fakeHandler{err: tt.err, applicationPanic: tt.panics})
					if tt.panics {
						require.NoError(t, err)
					} else {
						require.Error(t, err)
					}

					snap := root.Snapshot()
					successTags := newTags(_directionInbound, "")
					errTags := newTags(_directionInbound, tt.errName)

					// so we don't have create a sorting implementaion, manually place the
					// first two expected counter snapshots, based on the error fault.
					counters := make([]metrics.Snapshot, 0, 10)
					if tt.panics {
						counters = append(counters,
							metrics.Snapshot{Name: "calls", Tags: successTags, Value: 1},
							metrics.Snapshot{Name: "panics", Tags: successTags, Value: 1})
					} else if statusFault(yarpcerrors.FromError(tt.err)) == clientFault {
						counters = append(counters,
							metrics.Snapshot{Name: "caller_failures", Tags: errTags, Value: 1},
							metrics.Snapshot{Name: "calls", Tags: successTags, Value: 1},
							metrics.Snapshot{Name: "panics", Tags: successTags, Value: 0})
					} else {
						counters = append(counters,
							metrics.Snapshot{Name: "calls", Tags: successTags, Value: 1},
							metrics.Snapshot{Name: "panics", Tags: successTags, Value: 0},
							metrics.Snapshot{Name: "server_failures", Tags: errTags, Value: 1})
					}

					want := &metrics.RootSnapshot{
						// only the failure vector counters will have an error value passed
						// into tags()
						Counters: append(counters,
							metrics.Snapshot{Name: "stream_receive_successes", Tags: successTags, Value: 0},
							metrics.Snapshot{Name: "stream_receives", Tags: successTags, Value: 0},
							metrics.Snapshot{Name: "stream_send_successes", Tags: successTags, Value: 0},
							metrics.Snapshot{Name: "stream_sends", Tags: successTags, Value: 0},
							metrics.Snapshot{Name: "successes", Tags: successTags, Value: 1}),
						Gauges: []metrics.Snapshot{
							{Name: "streams_active", Tags: successTags, Value: 0},
						},
						Histograms: []metrics.HistogramSnapshot{
							{Name: "stream_duration_ms", Tags: successTags, Unit: time.Millisecond, Values: []int64{1}},
						},
					}
					assert.Equal(t, want, snap, "unexpected metrics snapshot")
				}
				if tt.panics {
					assert.Panics(t, handleStreamAndValidateFn)
				} else {
					assert.NotPanics(t, handleStreamAndValidateFn)
				}
			})
		}
	})

	t.Run("error server - send and recv", func(t *testing.T) {
		root := metrics.New()
		scope := root.Scope()
		mw := NewMiddleware(Config{
			Logger:           zap.NewNop(),
			Scope:            scope,
			ContextExtractor: NewNopContextExtractor(),
		})

		sendErr := errors.New("send err")
		receiveErr := errors.New("receive err")

		stream, err := transport.NewServerStream(&fakeStream{
			request:    req,
			sendErr:    sendErr,
			receiveErr: receiveErr,
		})
		require.NoError(t, err)

		err = mw.HandleStream(stream, &fakeHandler{
			handleStream: func(stream *transport.ServerStream) {
				err := stream.SendMessage(context.Background(), nil /*message*/)
				require.Error(t, err)
				_, err = stream.ReceiveMessage(context.Background())
				require.Error(t, err)
			}})
		require.NoError(t, err)

		snap := root.Snapshot()
		successTags := newTags(_directionInbound, "")
		errTags := newTags(_directionInbound, "unknown_internal_yarpc")

		want := &metrics.RootSnapshot{
			Counters: []metrics.Snapshot{
				{Name: "calls", Tags: successTags, Value: 1},
				{Name: "panics", Tags: successTags, Value: 0},
				{Name: "stream_receive_failures", Tags: errTags, Value: 1},
				{Name: "stream_receive_successes", Tags: successTags, Value: 0},
				{Name: "stream_receives", Tags: successTags, Value: 1},
				{Name: "stream_send_failures", Tags: errTags, Value: 1},
				{Name: "stream_send_successes", Tags: successTags, Value: 0},
				{Name: "stream_sends", Tags: successTags, Value: 1},
				{Name: "successes", Tags: successTags, Value: 1},
			},
			Gauges: []metrics.Snapshot{
				{Name: "streams_active", Tags: successTags, Value: 0}, // opened (+1) then closed (-1)
			},
			Histograms: []metrics.HistogramSnapshot{
				{Name: "stream_duration_ms", Tags: successTags, Unit: time.Millisecond, Values: []int64{1}},
			},
		}
		assert.Equal(t, want, snap, "unexpected metrics snapshot")
	})

	t.Run("success client", func(t *testing.T) {
		root := metrics.New()
		scope := root.Scope()
		mw := NewMiddleware(Config{
			Logger:           zap.NewNop(),
			Scope:            scope,
			ContextExtractor: NewNopContextExtractor(),
		})

		stream, err := mw.CallStream(context.Background(), req, fakeOutbound{})
		require.NoError(t, err)
		err = stream.SendMessage(context.Background(), nil /* message */)
		require.NoError(t, err)
		_, err = stream.ReceiveMessage(context.Background())
		require.NoError(t, err)
		require.NoError(t, stream.Close(context.Background()))

		snap := root.Snapshot()
		tags := newTags(_directionOutbound, "")

		// successful handshake, send, recv and close
		want := &metrics.RootSnapshot{
			Counters: []metrics.Snapshot{
				{Name: "calls", Tags: tags, Value: 1},
				{Name: "panics", Tags: tags, Value: 0},
				{Name: "stream_receive_successes", Tags: tags, Value: 1},
				{Name: "stream_receives", Tags: tags, Value: 1},
				{Name: "stream_send_successes", Tags: tags, Value: 1},
				{Name: "stream_sends", Tags: tags, Value: 1},
				{Name: "successes", Tags: tags, Value: 1},
			},
			Gauges: []metrics.Snapshot{
				{Name: "streams_active", Tags: tags, Value: 0}, // opened (+1) then closed (-1)
			},
			Histograms: []metrics.HistogramSnapshot{
				{Name: "stream_duration_ms", Tags: tags, Unit: time.Millisecond, Values: []int64{1}},
			},
		}
		assert.Equal(t, want, snap, "unexpected metrics snapshot")
	})

	t.Run("error client handshake", func(t *testing.T) {
		root := metrics.New()
		scope := root.Scope()
		mw := NewMiddleware(Config{
			Logger:           zap.NewNop(),
			Scope:            scope,
			ContextExtractor: NewNopContextExtractor(),
		})

		clientErr := errors.New("client err")
		_, err := mw.CallStream(context.Background(), req, fakeOutbound{err: clientErr})
		require.Error(t, err)

		snap := root.Snapshot()
		successTags := newTags(_directionOutbound, "")
		errTags := newTags(_directionOutbound, "unknown_internal_yarpc")

		want := &metrics.RootSnapshot{
			// only the failure vector counters will have an error value passed
			// into tags()
			Counters: []metrics.Snapshot{
				{Name: "calls", Tags: successTags, Value: 1},
				{Name: "panics", Tags: successTags, Value: 0},
				{Name: "server_failures", Tags: errTags, Value: 1},
				{Name: "stream_receive_successes", Tags: successTags, Value: 0},
				{Name: "stream_receives", Tags: successTags, Value: 0},
				{Name: "stream_send_successes", Tags: successTags, Value: 0},
				{Name: "stream_sends", Tags: successTags, Value: 0},
				{Name: "successes", Tags: successTags, Value: 0},
			},
			Gauges: []metrics.Snapshot{
				{Name: "streams_active", Tags: successTags, Value: 0},
			},
			Histograms: []metrics.HistogramSnapshot{
				{Name: "stream_duration_ms", Tags: successTags, Unit: time.Millisecond},
			},
		}
		assert.Equal(t, want, snap, "unexpected metrics snapshot")
	})

	t.Run("error client - send recv close", func(t *testing.T) {
		root := metrics.New()
		scope := root.Scope()
		mw := NewMiddleware(Config{
			Logger:           zap.NewNop(),
			Scope:            scope,
			ContextExtractor: NewNopContextExtractor(),
		})

		sendErr := errors.New("send err")
		receiveErr := errors.New("receive err")
		closeErr := errors.New("close err")

		stream, err := mw.CallStream(context.Background(), req, fakeOutbound{
			stream: fakeStream{
				sendErr:    sendErr,
				receiveErr: receiveErr,
				closeErr:   closeErr,
			}})
		require.NoError(t, err)

		err = stream.SendMessage(context.Background(), nil /* message */)
		require.Error(t, err)
		_, err = stream.ReceiveMessage(context.Background())
		require.Error(t, err)
		err = stream.Close(context.Background())
		require.Error(t, err)

		snap := root.Snapshot()
		successTags := newTags(_directionOutbound, "")
		errTags := newTags(_directionOutbound, "unknown_internal_yarpc")

		// successful handshake, send, recv and close
		want := &metrics.RootSnapshot{
			Counters: []metrics.Snapshot{
				{Name: "calls", Tags: successTags, Value: 1},
				{Name: "panics", Tags: successTags, Value: 0},
				{Name: "server_failures", Tags: errTags, Value: 1},
				{Name: "stream_receive_failures", Tags: errTags, Value: 1},
				{Name: "stream_receive_successes", Tags: successTags, Value: 0},
				{Name: "stream_receives", Tags: successTags, Value: 1},
				{Name: "stream_send_failures", Tags: errTags, Value: 1},
				{Name: "stream_send_successes", Tags: successTags, Value: 0},
				{Name: "stream_sends", Tags: successTags, Value: 1},
				{Name: "successes", Tags: successTags, Value: 1},
			},
			Gauges: []metrics.Snapshot{
				{Name: "streams_active", Tags: successTags, Value: 0}, // opened (+1) then closed (-1)
			},
			Histograms: []metrics.HistogramSnapshot{
				{Name: "stream_duration_ms", Tags: successTags, Unit: time.Millisecond, Values: []int64{1}},
			},
		}
		assert.Equal(t, want, snap, "unexpected metrics snapshot")
	})
}
