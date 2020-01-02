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

package yarpc_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	. "go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/introspection"
	"go.uber.org/yarpc/internal/observability"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
	tchannelgo "github.com/uber/tchannel-go"
	"go.uber.org/atomic"
	"go.uber.org/multierr"
	thriftrwversion "go.uber.org/thriftrw/version"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func basicConfig(t testing.TB) Config {
	httpTransport := http.NewTransport()
	tchannelTransport, err := tchannel.NewChannelTransport(tchannel.ServiceName("test"))
	require.NoError(t, err)

	return Config{
		Name: "test",
		Inbounds: Inbounds{
			tchannelTransport.NewInbound(),
			httpTransport.NewInbound(":0"),
		},
	}
}

func outboundConfig(t testing.TB) Config {
	cfg := basicConfig(t)
	cfg.Outbounds = Outbounds{"my-test-service": {
		Unary: http.NewTransport().NewSingleOutbound("http://127.0.0.1:1234"),
	}}
	return cfg
}

func basicDispatcher(t testing.TB) *Dispatcher {
	return NewDispatcher(basicConfig(t))
}

func TestDispatcherNamePanic(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
	}{
		{
			name: "no service name",
		},
		{
			name:        "invalid service name",
			serviceName: "--",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Panics(t, func() {
				NewDispatcher(Config{Name: tt.serviceName})
			},
				"expected to panic")
		})
	}
}

func TestDispatcherRegisterPanic(t *testing.T) {
	d := basicDispatcher(t)

	require.Panics(t, func() {
		d.Register([]transport.Procedure{
			{
				HandlerSpec: transport.HandlerSpec{},
			},
		})
	}, "expected unknown handler type to panic")
}

func TestInboundsReturnsACopy(t *testing.T) {
	dispatcher := basicDispatcher(t)

	inbounds := dispatcher.Inbounds()
	require.Len(t, inbounds, 2, "expected two inbounds")
	assert.NotNil(t, inbounds[0], "must not be nil")
	assert.NotNil(t, inbounds[1], "must not be nil")

	// Mutate the list and verify that the next call still returns non-nil
	// results.
	inbounds[0] = nil
	inbounds[1] = nil

	inbounds = dispatcher.Inbounds()
	require.Len(t, inbounds, 2, "expected two inbounds")
	assert.NotNil(t, inbounds[0], "must not be nil")
	assert.NotNil(t, inbounds[1], "must not be nil")
}

func TestInboundsOrderIsMaintained(t *testing.T) {
	dispatcher := basicDispatcher(t)

	// Order must be maintained
	_, ok := dispatcher.Inbounds()[0].(*tchannel.ChannelInbound)
	assert.True(t, ok, "first inbound must be TChannel")

	_, ok = dispatcher.Inbounds()[1].(*http.Inbound)
	assert.True(t, ok, "second inbound must be HTTP")
}

func TestInboundsOrderAfterStart(t *testing.T) {
	dispatcher := basicDispatcher(t)

	require.NoError(t, dispatcher.Start(), "failed to start Dispatcher")
	defer dispatcher.Stop()

	inbounds := dispatcher.Inbounds()

	tchInbound := inbounds[0].(*tchannel.ChannelInbound)
	assert.NotEqual(t, "0.0.0.0:0", tchInbound.Channel().PeerInfo().HostPort)

	httpInbound := inbounds[1].(*http.Inbound)
	assert.NotNil(t, httpInbound.Addr(), "expected an HTTP addr")
}

func TestOutboundsReturnsACopy(t *testing.T) {
	testService := "my-test-service"
	d := NewDispatcher(Config{
		Name: "test",
		Outbounds: Outbounds{
			testService: {
				Unary: http.NewTransport().NewSingleOutbound("http://127.0.0.1:1234"),
			},
		},
	})

	outbounds := d.Outbounds()
	require.Len(t, outbounds, 1, "expected one outbound")
	assert.Contains(t, outbounds, testService, "must contain my-test-service")

	// Mutate the map and verify that the next call still returns non-nil
	// results.
	delete(outbounds, "my-test-service")

	outbounds = d.Outbounds()
	require.Len(t, outbounds, 1, "expected one outbound")
	assert.Contains(t, outbounds, testService, "must contain my-test-service")
}

func TestStartStopFailures(t *testing.T) {
	tests := []struct {
		desc string

		inbounds   func(*gomock.Controller) Inbounds
		outbounds  func(*gomock.Controller) Outbounds
		procedures func(*gomock.Controller) []transport.Procedure

		wantStartErr string
		wantStopErr  string
	}{
		{
			desc: "all success",
			inbounds: func(mockCtrl *gomock.Controller) Inbounds {
				inbounds := make(Inbounds, 10)
				for i := range inbounds {
					in := transporttest.NewMockInbound(mockCtrl)
					in.EXPECT().Transports()
					in.EXPECT().SetRouter(gomock.Any())
					in.EXPECT().Start().Return(nil)
					in.EXPECT().Stop().Return(nil)
					inbounds[i] = in
				}
				return inbounds
			},
			outbounds: func(mockCtrl *gomock.Controller) Outbounds {
				outbounds := make(Outbounds, 10)
				for i := 0; i < 10; i++ {
					out := transporttest.NewMockUnaryOutbound(mockCtrl)
					out.EXPECT().Transports()
					out.EXPECT().Start().Return(nil)
					out.EXPECT().Stop().Return(nil)
					outbounds[fmt.Sprintf("service-%v", i)] =
						transport.Outbounds{
							Unary: out,
						}
				}
				return outbounds
			},
		},
		{
			desc: "all success streaming",
			inbounds: func(mockCtrl *gomock.Controller) Inbounds {
				inbounds := make(Inbounds, 10)
				for i := range inbounds {
					in := transporttest.NewMockInbound(mockCtrl)
					in.EXPECT().Transports()
					in.EXPECT().SetRouter(gomock.Any())
					in.EXPECT().Start().Return(nil)
					in.EXPECT().Stop().Return(nil)
					inbounds[i] = in
				}
				return inbounds
			},
			outbounds: func(mockCtrl *gomock.Controller) Outbounds {
				outbounds := make(Outbounds, 10)
				for i := 0; i < 10; i++ {
					out := transporttest.NewMockStreamOutbound(mockCtrl)
					out.EXPECT().Transports()
					out.EXPECT().Start().Return(nil)
					out.EXPECT().Stop().Return(nil)
					outbounds[fmt.Sprintf("service-%v", i)] =
						transport.Outbounds{
							Stream: out,
						}
				}
				return outbounds
			},
			procedures: func(mockCtrl *gomock.Controller) []transport.Procedure {
				proc := transport.Procedure{
					Name:        "test",
					HandlerSpec: transport.NewStreamHandlerSpec(transporttest.NewMockStreamHandler(mockCtrl)),
				}
				return []transport.Procedure{proc}
			},
		},
		{
			desc: "inbound 6 start failure",
			inbounds: func(mockCtrl *gomock.Controller) Inbounds {
				inbounds := make(Inbounds, 10)
				for i := range inbounds {
					in := transporttest.NewMockInbound(mockCtrl)
					in.EXPECT().Transports()
					in.EXPECT().SetRouter(gomock.Any())
					if i == 6 {
						in.EXPECT().Start().Return(errors.New("great sadness"))
					} else {
						in.EXPECT().Start().Return(nil)
						in.EXPECT().Stop().Return(nil)
					}
					inbounds[i] = in
				}
				return inbounds
			},
			outbounds: func(mockCtrl *gomock.Controller) Outbounds {
				outbounds := make(Outbounds, 10)
				for i := 0; i < 10; i++ {
					out := transporttest.NewMockUnaryOutbound(mockCtrl)
					out.EXPECT().Transports()
					out.EXPECT().Start().Return(nil)
					out.EXPECT().Stop().Return(nil)
					outbounds[fmt.Sprintf("service-%v", i)] =
						transport.Outbounds{
							Unary: out,
						}
				}
				return outbounds
			},
			wantStartErr: "great sadness",
		},
		{
			desc: "inbound 7 stop failure",
			inbounds: func(mockCtrl *gomock.Controller) Inbounds {
				inbounds := make(Inbounds, 10)
				for i := range inbounds {
					in := transporttest.NewMockInbound(mockCtrl)
					in.EXPECT().Transports()
					in.EXPECT().SetRouter(gomock.Any())
					in.EXPECT().Start().Return(nil)
					if i == 7 {
						in.EXPECT().Stop().Return(errors.New("great sadness"))
					} else {
						in.EXPECT().Stop().Return(nil)
					}
					inbounds[i] = in
				}
				return inbounds
			},
			outbounds: func(mockCtrl *gomock.Controller) Outbounds {
				outbounds := make(Outbounds, 10)
				for i := 0; i < 10; i++ {
					out := transporttest.NewMockUnaryOutbound(mockCtrl)
					out.EXPECT().Transports()
					out.EXPECT().Start().Return(nil)
					out.EXPECT().Stop().Return(nil)
					outbounds[fmt.Sprintf("service-%v", i)] =
						transport.Outbounds{
							Unary: out,
						}
				}
				return outbounds
			},
			wantStopErr: "great sadness",
		},
		{
			desc: "outbound 5 start failure",
			inbounds: func(mockCtrl *gomock.Controller) Inbounds {
				inbounds := make(Inbounds, 10)
				for i := range inbounds {
					in := transporttest.NewMockInbound(mockCtrl)
					in.EXPECT().Transports()
					in.EXPECT().SetRouter(gomock.Any())
					in.EXPECT().Start().Times(0)
					in.EXPECT().Stop().Times(0)
					inbounds[i] = in
				}
				return inbounds
			},
			outbounds: func(mockCtrl *gomock.Controller) Outbounds {
				outbounds := make(Outbounds, 10)
				for i := 0; i < 10; i++ {
					out := transporttest.NewMockUnaryOutbound(mockCtrl)
					out.EXPECT().Transports()
					if i == 5 {
						out.EXPECT().Start().Return(errors.New("something went wrong"))
					} else {
						out.EXPECT().Start().Return(nil)
						out.EXPECT().Stop().Return(nil)
					}
					outbounds[fmt.Sprintf("service-%v", i)] =
						transport.Outbounds{
							Unary: out,
						}
				}
				return outbounds
			},
			wantStartErr: "something went wrong",
			// TODO: Include the name of the outbound in the error message
		},
		{
			desc: "inbound 7 stop failure",
			inbounds: func(mockCtrl *gomock.Controller) Inbounds {
				inbounds := make(Inbounds, 10)
				for i := range inbounds {
					in := transporttest.NewMockInbound(mockCtrl)
					in.EXPECT().Transports()
					in.EXPECT().SetRouter(gomock.Any())
					in.EXPECT().Start().Return(nil)
					in.EXPECT().Stop().Return(nil)
					inbounds[i] = in
				}
				return inbounds
			},
			outbounds: func(mockCtrl *gomock.Controller) Outbounds {
				outbounds := make(Outbounds, 10)
				for i := 0; i < 10; i++ {
					out := transporttest.NewMockUnaryOutbound(mockCtrl)
					out.EXPECT().Transports()
					out.EXPECT().Start().Return(nil)
					if i == 7 {
						out.EXPECT().Stop().Return(errors.New("something went wrong"))
					} else {
						out.EXPECT().Stop().Return(nil)
					}
					outbounds[fmt.Sprintf("service-%v", i)] =
						transport.Outbounds{
							Unary: out,
						}
				}
				return outbounds
			},
			wantStopErr: "something went wrong",
			// TODO: Include the name of the outbound in the error message
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			dispatcher := NewDispatcher(Config{
				Name:      "test",
				Inbounds:  tt.inbounds(mockCtrl),
				Outbounds: tt.outbounds(mockCtrl),
			})

			if tt.procedures != nil {
				dispatcher.Register(tt.procedures(mockCtrl))
			}

			err := dispatcher.Start()
			if tt.wantStartErr != "" {
				if assert.Error(t, err, "expected Start() to fail") {
					assert.Contains(t, err.Error(), tt.wantStartErr)
				}
				return
			}
			if !assert.NoError(t, err, "expected Start() to succeed") {
				return
			}

			err = dispatcher.Stop()
			if tt.wantStopErr == "" {
				assert.NoError(t, err, "expected Stop() to succeed")
				return
			}
			if assert.Error(t, err, "expected Stop() to fail") {
				assert.Contains(t, err.Error(), tt.wantStopErr)
			}
		})
	}
}

func TestPhasedStartStop(t *testing.T) {
	t.Run("in order", func(t *testing.T) {
		d := NewDispatcher(outboundConfig(t))
		starter, err := d.PhasedStart()
		require.NoError(t, err, "constructing phased starter failed")
		startErr := multierr.Combine(
			starter.StartTransports(),
			starter.StartOutbounds(),
			starter.StartInbounds(),
		)
		require.NoError(t, startErr, "phased startup failed")
		stopper, err := d.PhasedStop()
		require.NoError(t, err, "constructing phased stopped failed")
		stopErr := multierr.Combine(
			stopper.StopInbounds(),
			stopper.StopOutbounds(),
			stopper.StopTransports(),
		)
		require.NoError(t, stopErr, "phased shutdown failed")
	})

	t.Run("start out of order", func(t *testing.T) {
		d := NewDispatcher(outboundConfig(t))
		starter, err := d.PhasedStart()
		require.NoError(t, err, "constructing phased starter failed")

		// Must start transports first.
		assert.Error(t, starter.StartInbounds(), "succeeded inbounds before transports")
		assert.Error(t, starter.StartOutbounds(), "succeeded started outbounds before transports")
		require.NoError(t, starter.StartTransports(), "starting transports failed")

		// Must start outbounds second.
		assert.Error(t, starter.StartTransports(), "succeeded starting transports again")
		assert.Error(t, starter.StartInbounds(), "succeeded started inbounds before outbounds")
		require.NoError(t, starter.StartOutbounds(), "starting outbounds failed")

		// Must start inbounds last.
		assert.Error(t, starter.StartTransports(), "succeeded starting transports again")
		assert.Error(t, starter.StartOutbounds(), "succeeded starting outbounds again")
		require.NoError(t, starter.StartInbounds(), "starting inbounds failed")

		assert.NoError(t, d.Stop(), "shutting down dispatcher failed")
	})

	t.Run("stop out of order", func(t *testing.T) {
		d := NewDispatcher(outboundConfig(t))
		require.NoError(t, d.Start(), "starting dispatcher failed")

		stopper, err := d.PhasedStop()
		require.NoError(t, err, "constructing phased stopper failed")

		// Must stop inbounds first.
		assert.Error(t, stopper.StopTransports(), "succeeded stopping transports before inbounds")
		assert.Error(t, stopper.StopOutbounds(), "succeeded stopping outbounds before inbounds")
		require.NoError(t, stopper.StopInbounds(), "stopping inbunds failed")

		// Must stop outbounds second.
		assert.Error(t, stopper.StopInbounds(), "succeeded stopping inbounds again")
		assert.Error(t, stopper.StopTransports(), "succeeded stopping transports before outbounds")
		require.NoError(t, stopper.StopOutbounds(), "stopping outbounds failed")

		// Must stop transports last.
		assert.Error(t, stopper.StopInbounds(), "succeeded stopping inbounds again")
		assert.Error(t, stopper.StopOutbounds(), "succeeded stopping outbounds again")
		require.NoError(t, stopper.StopTransports(), "stopping transports failed")
	})
}

func TestPhasedStartRaces(t *testing.T) {
	d := NewDispatcher(outboundConfig(t))
	starter, err := d.PhasedStart()
	require.NoError(t, err, "constructing phased starter failed")

	const concurrency = 100
	run := make(chan struct{})
	errs := atomic.NewInt64(0)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-run
			if err := starter.StartTransports(); err != nil {
				errs.Inc()
			}
			if err := starter.StartOutbounds(); err != nil {
				errs.Inc()
			}
			if err := starter.StartInbounds(); err != nil {
				errs.Inc()
			}
		}()
	}
	close(run)
	wg.Wait()
	// Expect repeat calls to Start* to fail.
	assert.Equal(t, 3*concurrency-3, int(errs.Load()), "wrong number of errors")
	require.NoError(t, d.Stop(), "failed to cleanly shut down dispatcher")
}

func TestPhasedStopRaces(t *testing.T) {
	d := NewDispatcher(outboundConfig(t))
	require.NoError(t, d.Start(), "starting dispatcher failed")
	stopper, err := d.PhasedStop()
	require.NoError(t, err, "constructing phased stopper failed")

	const concurrency = 100
	run := make(chan struct{})
	errs := atomic.NewInt64(0)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-run
			if err := stopper.StopInbounds(); err != nil {
				errs.Inc()
			}
			if err := stopper.StopOutbounds(); err != nil {
				errs.Inc()
			}
			if err := stopper.StopTransports(); err != nil {
				errs.Inc()
			}
		}()
	}
	close(run)
	wg.Wait()
	// Expect repeat calls to Stop* to fail.
	assert.Equal(t, 3*concurrency-3, int(errs.Load()), "wrong number of errors")
}

func TestNoOutboundsForService(t *testing.T) {
	defer func() {
		r := recover()
		require.NotNil(t, r, "did not panic")
		assert.Equal(t, r, `no outbound set for outbound key "my-test-service" in dispatcher`)
	}()

	NewDispatcher(Config{
		Name: "test",
		Outbounds: Outbounds{
			"my-test-service": {},
		},
	})
}

func TestClientConfig(t *testing.T) {
	dispatcher := NewDispatcher(Config{
		Name: "test",
		Outbounds: Outbounds{
			"my-test-service": {
				Unary: http.NewTransport().NewSingleOutbound("http://127.0.0.1:1234"),
			},
		},
	})

	cc := dispatcher.ClientConfig("my-test-service")

	assert.Equal(t, "test", cc.Caller())
	assert.Equal(t, "my-test-service", cc.Service())
}

func TestClientConfigError(t *testing.T) {
	dispatcher := NewDispatcher(Config{
		Name: "test",
		Outbounds: Outbounds{
			"my-test-service": {
				Unary: http.NewTransport().NewSingleOutbound("http://127.0.0.1:1234"),
			},
		},
	})

	assert.Panics(t, func() { dispatcher.ClientConfig("wrong test name") })
}

func TestOutboundConfig(t *testing.T) {
	dispatcher := NewDispatcher(Config{
		Name: "test",
		Outbounds: Outbounds{
			"my-test-service": {
				Unary: http.NewTransport().NewSingleOutbound("http://127.0.0.1:1234"),
			},
		},
	})

	cc := dispatcher.MustOutboundConfig("my-test-service")
	assert.Equal(t, "test", cc.CallerName)
	assert.Equal(t, "my-test-service", cc.Outbounds.ServiceName)
}

func TestOutboundConfigError(t *testing.T) {
	dispatcher := NewDispatcher(Config{
		Name: "test",
		Outbounds: Outbounds{
			"my-test-service": {
				Unary: http.NewTransport().NewSingleOutbound("http://127.0.0.1:1234"),
			},
		},
	})

	assert.Panics(t, func() { dispatcher.MustOutboundConfig("wrong test name") })
	oc, ok := dispatcher.OutboundConfig("wrong test name")
	assert.False(t, ok, "getting outbound config should not have succeeded")
	assert.Nil(t, oc, "getting outbound config should not have succeeded")
}

func TestInboundMiddleware(t *testing.T) {
	dispatcher := NewDispatcher(Config{
		Name: "test",
	})

	mw := dispatcher.InboundMiddleware()

	assert.NotNil(t, mw)
}

func TestClientConfigWithOutboundServiceNameOverride(t *testing.T) {
	dispatcher := NewDispatcher(Config{
		Name: "test",
		Outbounds: Outbounds{
			"my-test-service": {
				ServiceName: "my-real-service",
				Unary:       http.NewTransport().NewSingleOutbound("http://127.0.0.1:1234"),
			},
		},
	})

	cc := dispatcher.ClientConfig("my-test-service")

	assert.Equal(t, "test", cc.Caller())
	assert.Equal(t, "my-real-service", cc.Service())
}

func TestEnableObservabilityMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req := &transport.Request{
		Service:   "test",
		Caller:    "test",
		Procedure: "test",
		Encoding:  transport.Encoding("test"),
	}
	out := transporttest.NewMockUnaryOutbound(mockCtrl)
	out.EXPECT().Transports().AnyTimes()
	out.EXPECT().Call(ctx, req).Times(1).Return(nil, nil)

	core, logs := observer.New(zapcore.DebugLevel)
	dispatcher := NewDispatcher(Config{
		Name: "test",
		Outbounds: Outbounds{
			"my-test-service": {
				ServiceName: "my-real-service",
				Unary:       out,
			},
		},
		Logging: LoggingConfig{
			Zap: zap.New(core),
		},
		DisableAutoObservabilityMiddleware: false,
	})

	cc := dispatcher.MustOutboundConfig("my-test-service")
	_, err := cc.Outbounds.Unary.Call(ctx, req)
	require.NoError(t, err)

	// There should be one log.
	assert.Equal(t, 1, logs.Len())
}

func TestObservabilityMiddlewareApplicationErrorLevel(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req := &transport.Request{
		Service:   "test",
		Caller:    "test",
		Procedure: "test",
		Encoding:  transport.Encoding("test"),
	}
	out := transporttest.NewMockUnaryOutbound(mockCtrl)
	out.EXPECT().Transports().AnyTimes()
	out.EXPECT().Call(ctx, req).Return(&transport.Response{ApplicationError: true}, nil)

	core, logs := observer.New(zapcore.DebugLevel)

	infoLevel := zapcore.InfoLevel
	dispatcher := NewDispatcher(Config{
		Name: "test",
		Outbounds: Outbounds{
			"my-test-service": {
				ServiceName: "my-real-service",
				Unary:       out,
			},
		},
		Logging: LoggingConfig{
			Zap: zap.New(core),
			Levels: LogLevelConfig{
				ApplicationError: &infoLevel,
			},
		},
	})

	cc := dispatcher.MustOutboundConfig("my-test-service")
	_, err := cc.Outbounds.Unary.Call(ctx, req)
	require.NoError(t, err)

	assert.Equal(t, 1, logs.Len())
	e := logs.TakeAll()[0]
	assert.Equal(t, zapcore.InfoLevel, e.Level)
	assert.Equal(t, "Error making outbound call.", e.Message)

}

func TestDisableObservabilityMiddleware(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req := &transport.Request{
		Service:   "test",
		Caller:    "test",
		Procedure: "test",
		Encoding:  transport.Encoding("test"),
	}
	out := transporttest.NewMockUnaryOutbound(mockCtrl)
	out.EXPECT().Transports().AnyTimes()
	out.EXPECT().Call(ctx, req).Times(1).Return(nil, nil)

	core, logs := observer.New(zapcore.DebugLevel)
	dispatcher := NewDispatcher(Config{
		Name: "test",
		Outbounds: Outbounds{
			"my-test-service": {
				ServiceName: "my-real-service",
				Unary:       out,
			},
		},
		Logging: LoggingConfig{
			Zap: zap.New(core),
		},
		DisableAutoObservabilityMiddleware: true,
	})

	cc := dispatcher.MustOutboundConfig("my-test-service")
	_, err := cc.Outbounds.Unary.Call(ctx, req)
	require.NoError(t, err)

	// There should be no logs.
	assert.Equal(t, 0, logs.Len())
}

func TestObservabilityConfig(t *testing.T) {
	// Validate that we can start a dispatcher with various logging and metrics
	// configs.
	logCfgs := []LoggingConfig{
		{},
		{Zap: zap.NewNop()},
		{ContextExtractor: observability.NewNopContextExtractor()},
		{Zap: zap.NewNop(), ContextExtractor: observability.NewNopContextExtractor()},
	}
	metricsCfgs := []MetricsConfig{
		{},
		{Tally: tally.NewTestScope("" /* prefix */, nil /* tags */)},
	}

	for _, l := range logCfgs {
		for _, m := range metricsCfgs {
			cfg := basicConfig(t)
			cfg.Logging = l
			cfg.Metrics = m
			assert.NotPanics(
				t,
				func() { NewDispatcher(cfg) },
				"Failed to create dispatcher with config %+v.", cfg,
			)
		}
	}
}

func TestIntrospect(t *testing.T) {
	httpTransport := http.NewTransport()
	tchannelChannelTransport, err := tchannel.NewChannelTransport(tchannel.ServiceName("test"), tchannel.ListenAddr(":4040"))
	require.NoError(t, err)
	tchannelTransport, err := tchannel.NewTransport(tchannel.ServiceName("test"), tchannel.ListenAddr(":5050"))
	require.NoError(t, err)
	httpOutbound := httpTransport.NewSingleOutbound("http://127.0.0.1:1234")

	config := Config{
		Name: "test",
		Inbounds: Inbounds{
			httpTransport.NewInbound(":0"),
			tchannelChannelTransport.NewInbound(),
			tchannelTransport.NewInbound(),
		},
		Outbounds: Outbounds{
			"test-client-http": {
				Unary:  httpOutbound,
				Oneway: httpOutbound,
			},
			"test-client-tchannel-channel": {
				Unary: tchannelChannelTransport.NewSingleOutbound("127.0.0.1:2345"),
			},
			"test-client-tchannel": {
				Unary: tchannelTransport.NewSingleOutbound("127.0.0.1:3456"),
			},
		},
	}
	dispatcher := NewDispatcher(config)

	dispatcherStatus := dispatcher.Introspect()

	assert.Equal(t, config.Name, dispatcherStatus.Name)
	assert.NotEmpty(t, dispatcherStatus.ID)
	assert.Empty(t, dispatcherStatus.Procedures)
	assert.Len(t, dispatcherStatus.Inbounds, 3)
	assert.Len(t, dispatcherStatus.Outbounds, 4)

	inboundStatus := getInboundStatus(t, dispatcherStatus.Inbounds, "http", "")
	assert.Equal(t, "Stopped", inboundStatus.State)
	inboundStatus = getInboundStatus(t, dispatcherStatus.Inbounds, "tchannel", ":4040")
	assert.Equal(t, "ChannelClient", inboundStatus.State)
	inboundStatus = getInboundStatus(t, dispatcherStatus.Inbounds, "tchannel", ":5050")
	assert.Equal(t, "", inboundStatus.State)

	outboundStatus := getOutboundStatus(t, dispatcherStatus.Outbounds, "test-client-http", "unary")
	assert.Equal(t, "http://127.0.0.1:1234", outboundStatus.Endpoint)
	assert.Equal(t, "Stopped", outboundStatus.State)
	assert.Equal(t, "test-client-http", outboundStatus.OutboundKey)
	outboundStatus = getOutboundStatus(t, dispatcherStatus.Outbounds, "test-client-http", "oneway")
	assert.Equal(t, "http://127.0.0.1:1234", outboundStatus.Endpoint)
	assert.Equal(t, "Stopped", outboundStatus.State)
	assert.Equal(t, "test-client-http", outboundStatus.OutboundKey)
	outboundStatus = getOutboundStatus(t, dispatcherStatus.Outbounds, "test-client-tchannel-channel", "unary")
	assert.Equal(t, "127.0.0.1:2345", outboundStatus.Endpoint)
	assert.Equal(t, "Stopped", outboundStatus.State)
	assert.Equal(t, "test-client-tchannel-channel", outboundStatus.OutboundKey)
	outboundStatus = getOutboundStatus(t, dispatcherStatus.Outbounds, "test-client-tchannel", "unary")
	assert.Equal(t, "Stopped", outboundStatus.State)
	assert.Equal(t, "test-client-tchannel", outboundStatus.OutboundKey)

	packageNameToVersion := make(map[string]string, len(dispatcherStatus.PackageVersions))
	for _, packageVersion := range dispatcherStatus.PackageVersions {
		assert.Empty(t, packageNameToVersion[packageVersion.Name])
		packageNameToVersion[packageVersion.Name] = packageVersion.Version
	}
	checkPackageVersion(t, packageNameToVersion, "yarpc", Version)
	checkPackageVersion(t, packageNameToVersion, "tchannel", tchannelgo.VersionInfo)
	checkPackageVersion(t, packageNameToVersion, "thriftrw", thriftrwversion.Version)
	checkPackageVersion(t, packageNameToVersion, "go", runtime.Version())
}

func getInboundStatus(t *testing.T, inbounds []introspection.InboundStatus, transport string, endpoint string) introspection.InboundStatus {
	for _, inboundStatus := range inbounds {
		if inboundStatus.Transport == transport && inboundStatus.Endpoint == endpoint {
			return inboundStatus
		}
	}
	t.Fatalf("could not find inbound with transport %s and endpoint %s", transport, endpoint)
	return introspection.InboundStatus{}
}

func getOutboundStatus(t *testing.T, outbounds []introspection.OutboundStatus, service string, rpcType string) introspection.OutboundStatus {
	for _, outboundStatus := range outbounds {
		if outboundStatus.Service == service && outboundStatus.RPCType == rpcType {
			return outboundStatus
		}
	}
	t.Fatalf("could not find outbound with service %s and rpcType %s", service, rpcType)
	return introspection.OutboundStatus{}
}

func checkPackageVersion(t *testing.T, packageNameToVersion map[string]string, key string, expectedVersion string) {
	version := packageNameToVersion[key]
	assert.NotEmpty(t, version)
	assert.Equal(t, expectedVersion, version)
}
