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
	"io"
	"strings"
	"sync"
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

	rawErr := errors.New("fail")
	yErrNoDetails := yarpcerrors.Newf(yarpcerrors.CodeAborted, "fail")
	yErrWithDetails := yarpcerrors.Newf(yarpcerrors.CodeAborted, "fail").WithDetails([]byte("err detail"))
	yErrResourceExhausted := yarpcerrors.CodeResourceExhausted
	appErrDetails := "an app error detail string, usually from thriftEx.Error()!"

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
		desc                  string
		err                   error             // downstream error
		applicationErr        bool              // downstream application error
		applicationErrName    string            // downstream application error name
		applicationErrDetails string            // downstream application error message
		applicationErrCode    *yarpcerrors.Code // downstream application error code
		wantErrLevel          zapcore.Level
		wantInboundMsg        string
		wantOutboundMsg       string
		wantFields            []zapcore.Field
	}

	tests := []test{
		{
			desc:            "success",
			wantErrLevel:    zapcore.InfoLevel,
			wantInboundMsg:  "Handled inbound request.",
			wantOutboundMsg: "Made outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", true),
				zap.Skip(), // ContextExtractor
			},
		},
		{
			desc:            "downstream transport error",
			err:             rawErr,
			wantErrLevel:    zapcore.ErrorLevel,
			wantInboundMsg:  "Error handling inbound request.",
			wantOutboundMsg: "Error making outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Skip(),
				zap.Error(rawErr),
				zap.String(_errorCodeLogKey, "unknown"),
			},
		},
		{
			desc:                  "thrift application error with no name",
			applicationErr:        true,
			applicationErrDetails: appErrDetails,
			wantErrLevel:          zapcore.WarnLevel,
			wantInboundMsg:        "Error handling inbound request.",
			wantOutboundMsg:       "Error making outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Skip(),
				zap.String("error", "application_error"),
				zap.String("errorDetails", appErrDetails),
			},
		},
		{
			desc:                  "thrift application error with name and code",
			applicationErr:        true,
			applicationErrName:    "FunkyThriftError",
			applicationErrDetails: appErrDetails,
			applicationErrCode:    &yErrResourceExhausted,
			wantErrLevel:          zapcore.WarnLevel,
			wantInboundMsg:        "Error handling inbound request.",
			wantOutboundMsg:       "Error making outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Skip(),
				zap.String("error", "application_error"),
				zap.String("errorCode", "resource-exhausted"),
				zap.String("errorName", "FunkyThriftError"),
				zap.String("errorDetails", appErrDetails),
			},
		},
		{
			// ie 'errors.New' return in Protobuf handler
			desc:            "err and app error",
			err:             rawErr,
			applicationErr:  true, // always true for Protobuf handler errors
			wantErrLevel:    zapcore.ErrorLevel,
			wantInboundMsg:  "Error handling inbound request.",
			wantOutboundMsg: "Error making outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Skip(),
				zap.Error(rawErr),
				zap.String(_errorCodeLogKey, "unknown"),
			},
		},
		{
			// ie 'yarpcerror' or 'protobuf.NewError` return in Protobuf handler
			desc:            "yarpcerror, app error",
			err:             yErrNoDetails,
			applicationErr:  true, // always true for Protobuf handler errors
			wantErrLevel:    zapcore.ErrorLevel,
			wantInboundMsg:  "Error handling inbound request.",
			wantOutboundMsg: "Error making outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Skip(),
				zap.Error(yErrNoDetails),
				zap.String(_errorCodeLogKey, "aborted"),
			},
		},
		{
			// ie 'protobuf.NewError' return in Protobuf handler
			desc:                  "yarpcerror, app error with name and code",
			err:                   yErrNoDetails,
			applicationErr:        true, // always true for Protobuf handler errors
			wantErrLevel:          zapcore.ErrorLevel,
			applicationErrDetails: appErrDetails,
			applicationErrName:    "MyErrMessageName",
			wantInboundMsg:        "Error handling inbound request.",
			wantOutboundMsg:       "Error making outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Skip(), // ContextExtractor
				zap.Error(yErrNoDetails),
				zap.String(_errorCodeLogKey, "aborted"),
				zap.String(_errorNameLogKey, "MyErrMessageName"),
				zap.String(_errorDetailsLogKey, appErrDetails),
			},
		},
		{
			// ie Protobuf error detail return in Protobuf handler
			desc:            "err details, app error",
			err:             yErrWithDetails,
			applicationErr:  true, // always true for Protobuf handler errors
			wantErrLevel:    zapcore.WarnLevel,
			wantInboundMsg:  "Error handling inbound request.",
			wantOutboundMsg: "Error making outbound call.",
			wantFields: []zapcore.Field{
				zap.Duration("latency", 0),
				zap.Bool("successful", false),
				zap.Skip(),
				zap.Error(yErrWithDetails),
				zap.String(_errorCodeLogKey, "aborted"),
			},
		},
	}

	newHandler := func(t test) fakeHandler {
		return fakeHandler{
			err:                   t.err,
			applicationErr:        t.applicationErr,
			applicationErrName:    t.applicationErrName,
			applicationErrDetails: t.applicationErrDetails,
			applicationErrCode:    t.applicationErrCode,
		}
	}

	newOutbound := func(t test) fakeOutbound {
		return fakeOutbound{
			err:                   t.err,
			applicationErr:        t.applicationErr,
			applicationErrName:    t.applicationErrName,
			applicationErrDetails: t.applicationErrDetails,
			applicationErrCode:    t.applicationErrCode,
		}
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

	// helper function to retrieve observed logs, asserting the expected number
	getLogs := func(t *testing.T, num int) []observer.LoggedEntry {
		logs := logs.TakeAll()
		require.Equal(t, num, len(logs), "expected exactly %d logs, got %v: %#v", num, len(logs), logs)

		var entries []observer.LoggedEntry
		for _, e := range logs {
			// zero the time for easy comparisons
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

	t.Run("EOF is a success with an error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		clientStream, serverStream, finish, err := transporttest.MessagePipe(ctx, req)
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			finish(mw.HandleStream(serverStream, &fakeHandler{
				// send and receive messages in the handler
				handleStream: func(stream *transport.ServerStream) {
					// echo loop
					for {
						msg, err := stream.ReceiveMessage(ctx)
						if err == io.EOF {
							return
						}
						err = stream.SendMessage(ctx, msg)
						if err == io.EOF {
							return
						}
					}
				},
			}))
			wg.Done()
		}()

		{
			err := clientStream.SendMessage(ctx, nil)
			require.NoError(t, err)
		}

		{
			msg, err := clientStream.ReceiveMessage(ctx)
			require.NoError(t, err)
			assert.Nil(t, msg)
		}

		require.NoError(t, clientStream.Close(ctx))

		wg.Wait()

		logFields := func(err error) []zapcore.Field {
			return newZapFields(
				zap.String("direction", string(_directionInbound)),
				zap.String("rpcType", "Streaming"),
				zap.Bool("successful", true),
				zap.Skip(), // context extractor
				zap.Error(err),
			)
		}

		wantLogs := []observer.LoggedEntry{
			{
				// open stream
				Entry: zapcore.Entry{
					Message: _successStreamOpen,
				},
				Context: logFields(nil),
			},
			{
				// receive message
				Entry: zapcore.Entry{
					Message: _successfulStreamReceive,
				},
				Context: logFields(nil),
			},
			{
				// send message
				Entry: zapcore.Entry{
					Message: _successfulStreamSend,
				},
				Context: logFields(nil),
			},
			{
				// receive message (EOF)
				Entry: zapcore.Entry{
					Message: _successfulStreamReceive,
				},
				Context: logFields(io.EOF),
			},
			{
				// close stream
				Entry: zapcore.Entry{
					Message: _successStreamClose,
				},
				Context: append(logFields(nil), zap.Duration("duration", 0)),
			},
		}

		// log 1 - open stream
		// log 2 - receive message
		// log 3 - send message
		// log 4 - receive message
		// log 5 - close stream
		gotLogs := getLogs(t, 5)
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

	yErrAlreadyExists := yarpcerrors.CodeAlreadyExists
	yErrCodeUnknown := yarpcerrors.CodeUnknown

	type failureTags struct {
		errorTag     string
		errorNameTag string
	}

	type test struct {
		desc               string
		err                error             // downstream error
		applicationErr     bool              // downstream application error
		applicationErrName string            // downstream application error name
		applicationErrCode *yarpcerrors.Code // downstream application error code
		wantCalls          int
		wantSuccesses      int
		wantCallerFailures map[failureTags]int
		wantServerFailures map[failureTags]int
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
			wantCallerFailures: map[failureTags]int{
				{
					errorTag:     yarpcerrors.CodeInvalidArgument.String(),
					errorNameTag: _notSet,
				}: 1,
			},
		},
		{
			desc:          "internal error",
			err:           yarpcerrors.Newf(yarpcerrors.CodeInternal, "test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantServerFailures: map[failureTags]int{
				{
					errorTag:     yarpcerrors.CodeInternal.String(),
					errorNameTag: _notSet,
				}: 1,
			},
		},
		{
			desc:          "unknown (unwrapped) error",
			err:           errors.New("test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantServerFailures: map[failureTags]int{
				{
					errorTag:     "unknown_internal_yarpc",
					errorNameTag: _notSet,
				}: 1,
			},
		},
		{
			desc:          "custom error code error",
			err:           yarpcerrors.Newf(yarpcerrors.Code(1000), "test"),
			wantCalls:     1,
			wantSuccesses: 0,
			wantServerFailures: map[failureTags]int{
				{
					errorTag:     "1000",
					errorNameTag: _notSet,
				}: 1,
			},
		},
		{
			desc:               "application error name with no code",
			wantCalls:          1,
			wantSuccesses:      0,
			applicationErr:     true,
			applicationErrName: "SomeError",
			wantCallerFailures: map[failureTags]int{
				{
					errorTag:     "application_error",
					errorNameTag: "SomeError",
				}: 1,
			},
		},
		{
			desc:               "application error name with YARPC code - caller failure",
			wantCalls:          1,
			wantSuccesses:      0,
			applicationErr:     true,
			applicationErrName: "SomeError",
			applicationErrCode: &yErrAlreadyExists,
			wantCallerFailures: map[failureTags]int{
				{
					errorTag:     "already-exists",
					errorNameTag: "SomeError",
				}: 1,
			},
		},
		{
			desc:               "application error name with YARPC code - server failure",
			wantCalls:          1,
			wantSuccesses:      0,
			applicationErr:     true,
			applicationErrName: "InternalServerPain",
			applicationErrCode: &yErrCodeUnknown,
			wantServerFailures: map[failureTags]int{
				{
					errorTag:     "unknown",
					errorNameTag: "InternalServerPain",
				}: 1,
			},
		},
		{
			desc:               "application error with YARPC code and empty name",
			wantCalls:          1,
			wantSuccesses:      0,
			applicationErr:     true,
			applicationErrName: "",
			applicationErrCode: &yErrAlreadyExists,
			wantCallerFailures: map[failureTags]int{
				{
					errorTag:     "already-exists",
					errorNameTag: _notSet,
				}: 1,
			},
		},
	}

	newHandler := func(t test) fakeHandler {
		return fakeHandler{
			err:                t.err,
			applicationErr:     t.applicationErr,
			applicationErrName: t.applicationErrName,
			applicationErrCode: t.applicationErrCode,
		}
	}

	newOutbound := func(t test) fakeOutbound {
		return fakeOutbound{
			err:                t.err,
			applicationErr:     t.applicationErr,
			applicationErrName: t.applicationErrName,
			applicationErrCode: t.applicationErrCode,
		}
	}

	for _, tt := range tests {
		validate := func(mw *Middleware, direction string, rpcType transport.Type) {
			key, free := getKey(req, direction, rpcType)
			edge := mw.graph.getEdge(key)
			free()
			assert.EqualValues(t, tt.wantCalls, edge.calls.Load(), "expected calls mismatch")
			assert.EqualValues(t, tt.wantSuccesses, edge.successes.Load(), "expected successes mismatch")
			assert.EqualValues(t, 0, edge.panics.Load(), "expected panics mismatch")
			for failureTags, num := range tt.wantCallerFailures {
				assert.EqualValues(t, num, edge.callerFailures.MustGet(
					_error, failureTags.errorTag,
					_errorNameMetricsKey, failureTags.errorNameTag,
				).Load(), "expected caller failures mismatch")
			}
			for failureTags, num := range tt.wantServerFailures {
				assert.EqualValues(t, num, edge.serverFailures.MustGet(
					_error, failureTags.errorTag,
					_errorNameMetricsKey, failureTags.errorNameTag,
				).Load(), "expected server failures mismatch")
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

	yErrAlreadyExists := yarpcerrors.CodeAlreadyExists

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
		zap.String("errorCode", "already-exists"),
		zap.String("errorName", "SomeFakeError"),
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
		fakeHandler{
			err:                nil,
			applicationErr:     true,
			applicationErrName: "SomeFakeError",
			applicationErrCode: &yErrAlreadyExists,
		},
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
	defer stubTimeWithTimeVal(time.Now())()
	ttlMs := int64(1000)
	root := metrics.New()
	meter := root.Scope()
	mw := NewMiddleware(Config{
		Logger:           zap.NewNop(),
		Scope:            meter,
		ContextExtractor: NewNopContextExtractor(),
	})

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond*time.Duration(ttlMs)))
	defer cancel()
	err := mw.Handle(
		ctx,
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
			{
				Name: "timeout_ttl_ms",
				Tags: tags,
				Unit: time.Millisecond,
			},
			{
				Name:   "ttl_ms",
				Tags:   tags,
				Unit:   time.Millisecond,
				Values: []int64{ttlMs},
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
		"error_name":       _notSet,
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
			{
				Name: "timeout_ttl_ms",
				Tags: tags,
				Unit: time.Millisecond,
			},
			{
				Name: "ttl_ms",
				Tags: tags,
				Unit: time.Millisecond,
			},
		},
	}
	assert.Equal(t, want, snap, "Unexpected snapshot of metrics.")
}

func TestMiddlewareFailureWithDeadlineExceededSnapshot(t *testing.T) {
	defer stubTimeWithTimeVal(time.Now())()

	ttlMs := int64(1000)
	root := metrics.New()
	meter := root.Scope()
	mw := NewMiddleware(Config{
		Logger:           zap.NewNop(),
		Scope:            meter,
		ContextExtractor: NewNopContextExtractor(),
	})
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond*time.Duration(ttlMs)))
	defer cancel()
	err := mw.Handle(
		ctx,
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
		fakeHandler{err: yarpcerrors.DeadlineExceededErrorf("test deadline"), applicationErr: false},
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
		"error":            "deadline-exceeded",
		"error_name":       _notSet,
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
			{
				Name:   "timeout_ttl_ms",
				Tags:   tags,
				Unit:   time.Millisecond,
				Values: []int64{ttlMs},
			},
			{
				Name:   "ttl_ms",
				Tags:   tags,
				Unit:   time.Millisecond,
				Values: []int64{ttlMs},
			},
		},
	}
	assert.Equal(t, want, snap, "Unexpected snapshot of metrics.")
}

func TestApplicationErrorSnapShot(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		errTag     string
		errNameTag string
		appErr     bool
		appErrName string
	}{
		{
			name:       "status", // eg error returned in transport middleware
			err:        yarpcerrors.Newf(yarpcerrors.CodeAlreadyExists, "foo exists!"),
			errTag:     "already-exists",
			errNameTag: _notSet,
		},
		{
			name:       "status and app error", // eg Protobuf handler returning yarpcerrors.Status
			err:        yarpcerrors.Newf(yarpcerrors.CodeAlreadyExists, "foo exists!"),
			errTag:     "already-exists",
			errNameTag: _notSet,
			appErr:     true,
		},
		{
			name:       "no status and app error", // eg Thrift exception
			err:        errors.New("foo-bar-baz"),
			errTag:     "application_error",
			errNameTag: "FakeError1",
			appErr:     true,
			appErrName: "FakeError1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer stubTimeWithTimeVal(time.Now())()

			ttlMs := int64(1000)
			root := metrics.New()
			meter := root.Scope()
			mw := NewMiddleware(Config{
				Logger: zap.NewNop(),
				Scope:  meter,
			})
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ttlMs)*time.Millisecond)
			defer cancel()
			err := mw.Handle(
				ctx,
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
					err:                tt.err,
					applicationErr:     tt.appErr,
					applicationErrName: tt.appErrName,
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
				"error_name":       tt.errNameTag,
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
					{
						Name: "timeout_ttl_ms",
						Tags: tags,
						Unit: time.Millisecond,
					},
					{
						Name:   "ttl_ms",
						Tags:   tags,
						Unit:   time.Millisecond,
						Values: []int64{ttlMs},
					},
				},
			}
			assert.Equal(t, want, snap, "Unexpected snapshot of metrics.")
		})
	}
}

func TestUnaryInboundApplicationPanics(t *testing.T) {
	var err error
	root := metrics.New()
	scope := root.Scope()
	mw := NewMiddleware(Config{
		Logger:           zap.NewNop(),
		Scope:            scope,
		ContextExtractor: NewNopContextExtractor(),
	})
	newTags := func(direction directionName, withErr string) metrics.Tags {
		tags := metrics.Tags{
			"dest":             "service",
			"direction":        string(direction),
			"encoding":         "raw",
			"procedure":        "procedure",
			"routing_delegate": "rd",
			"routing_key":      "rk",
			"rpc_type":         transport.Unary.String(),
			"source":           "caller",
			"transport":        "unknown",
		}
		if withErr != "" {
			tags["error"] = withErr
		}
		return tags
	}
	tags := newTags(_directionInbound, "")
	errTags := newTags(_directionInbound, "application_error")

	t.Run("Test panic in Handle", func(t *testing.T) {
		t.Skip() // This test flaps. https://github.com/yarpc/yarpc-go/issues/1882
		// Relevant bucket marked XXX below.

		// As our fake handler is mocked to panic in the call, test that the invocation panics
		assert.Panics(t, func() {
			err = mw.Handle(
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
				fakeHandler{applicationPanic: true},
			)
		})
		require.NoError(t, err)

		want := &metrics.RootSnapshot{
			Counters: []metrics.Snapshot{
				{Name: "caller_failures", Tags: errTags, Value: 1},
				{Name: "calls", Tags: tags, Value: 1},
				{Name: "panics", Tags: tags, Value: 1},
				{Name: "successes", Tags: tags, Value: 0},
			},
			Histograms: []metrics.HistogramSnapshot{
				{
					Name:   "caller_failure_latency_ms",
					Tags:   tags,
					Unit:   time.Millisecond,
					Values: []int64{1}, // XXX this test flaps mysteriously. This figure is sometimes higher.
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
		assert.Equal(t, want, root.Snapshot(), "unexpected metrics snapshot")
	})
}

func TestStreamingInboundApplicationPanics(t *testing.T) {
	root := metrics.New()
	scope := root.Scope()
	mw := NewMiddleware(Config{
		Logger:           zap.NewNop(),
		Scope:            scope,
		ContextExtractor: NewNopContextExtractor(),
	})
	stream, err := transport.NewServerStream(&fakeStream{
		request: &transport.StreamRequest{
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
		},
	})
	require.NoError(t, err)
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
	tags := newTags(_directionInbound, "")
	errTags := newTags(_directionInbound, "unknown_internal_yarpc")

	t.Run("Test panic in HandleStream", func(t *testing.T) {
		t.Skip() // This test flaps. https://github.com/yarpc/yarpc-go/issues/1882
		// Relevant bucket marked XXX below.

		// As our fake handler is mocked to panic in the call, test that the invocation panics
		assert.Panics(t, func() {
			err = mw.HandleStream(stream, &fakeHandler{applicationPanic: true})
		})
		require.NoError(t, err)

		want := &metrics.RootSnapshot{
			Counters: []metrics.Snapshot{
				{Name: "calls", Tags: tags, Value: 1},
				{Name: "panics", Tags: tags, Value: 1},
				{Name: "server_failures", Tags: errTags, Value: 1},
				{Name: "stream_receive_successes", Tags: tags, Value: 0},
				{Name: "stream_receives", Tags: tags, Value: 0},
				{Name: "stream_send_successes", Tags: tags, Value: 0},
				{Name: "stream_sends", Tags: tags, Value: 0},
				{Name: "successes", Tags: tags, Value: 1},
			},
			Gauges: []metrics.Snapshot{
				{Name: "streams_active", Tags: tags, Value: 0},
			},
			Histograms: []metrics.HistogramSnapshot{
				{Name: "stream_duration_ms", Tags: tags, Unit: time.Millisecond, Values: []int64{1}}, // XXX sometimes >1.
			},
		}
		assert.Equal(t, want, root.Snapshot(), "unexpected metrics snapshot")
	})

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

	newTags := func(direction directionName, withErr string, withCallerFailureErrName string) metrics.Tags {
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
		if withCallerFailureErrName != "" {
			tags[_errorNameMetricsKey] = withCallerFailureErrName
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
		tags := newTags(_directionInbound, "" /* withErr */, "" /* withCallerFailureErrName */)

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
			name       string
			err        error
			errName    string
			appErrName string
		}{
			{
				name:       "client fault",
				err:        yarpcerrors.InvalidArgumentErrorf("client err"),
				errName:    yarpcerrors.CodeInvalidArgument.String(),
				appErrName: _notSet,
			},
			{
				name:       "server fault",
				err:        yarpcerrors.InternalErrorf("server err"),
				errName:    yarpcerrors.CodeInternal.String(),
				appErrName: _notSet,
			},
			{
				name:       "unknown fault",
				err:        errors.New("unknown fault"),
				errName:    "unknown_internal_yarpc",
				appErrName: _notSet,
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
				err = mw.HandleStream(stream, &fakeHandler{err: tt.err})
				require.Error(t, err)

				snap := root.Snapshot()
				successTags := newTags(_directionInbound, "", "")
				errTags := newTags(_directionInbound, tt.errName, tt.appErrName)

				// so we don't have create a sorting implementation, manually place the
				// first two expected counter snapshots, based on the error fault.
				counters := make([]metrics.Snapshot, 0, 10)
				if faultFromCode(yarpcerrors.FromError(tt.err).Code()) == clientFault {
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
		successTags := newTags(_directionInbound, "", "")
		errTags := newTags(_directionInbound, "unknown_internal_yarpc", "")

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
		tags := newTags(_directionOutbound, "", "")

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
		successTags := newTags(_directionOutbound, "", "")
		errTags := newTags(_directionOutbound, "unknown_internal_yarpc", _notSet)

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
		successTags := newTags(_directionOutbound, "", "")
		errTags := newTags(_directionOutbound, "unknown_internal_yarpc", "")
		serverFailureTags := newTags(_directionOutbound, "unknown_internal_yarpc", _notSet)

		// successful handshake, send, recv and close
		want := &metrics.RootSnapshot{
			Counters: []metrics.Snapshot{
				{Name: "calls", Tags: successTags, Value: 1},
				{Name: "panics", Tags: successTags, Value: 0},
				{Name: "server_failures", Tags: serverFailureTags, Value: 1},
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

func TestNewWriterIsEmpty(t *testing.T) {
	code := yarpcerrors.CodeDataLoss

	// set all fields on the response writer
	w := newWriter(&transporttest.FakeResponseWriter{})
	require.NotNil(t, w, "writer must not be nil")

	w.SetApplicationError()
	w.SetApplicationErrorMeta(&transport.ApplicationErrorMeta{
		Details: "foo", Name: "bar", Code: &code,
	})
	w.free()

	w = newWriter(nil /*transport.ResponseWriter*/)
	require.NotNil(t, w, "writer must not be nil")
	assert.Equal(t, writer{}, *w,
		"expected empty writer, fields were likely not cleared in the pool")
}
