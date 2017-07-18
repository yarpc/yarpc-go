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

package yarpc_test

import (
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
	tchannelgo "github.com/uber/tchannel-go"
	thriftrwversion "go.uber.org/thriftrw/version"
	. "go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/observability"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
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

func basicDispatcher(t testing.TB) *Dispatcher {
	return NewDispatcher(basicConfig(t))
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

func TestStartStopFailures(t *testing.T) {
	tests := []struct {
		desc string

		inbounds  func(*gomock.Controller) Inbounds
		outbounds func(*gomock.Controller) Outbounds

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

	config := Config{
		Name: "test",
		Inbounds: Inbounds{
			httpTransport.NewInbound(":0"),
		},
		Outbounds: Outbounds{
			"test-client": {
				Unary:  httpTransport.NewSingleOutbound("http://127.0.0.1:1234"),
				Oneway: httpTransport.NewSingleOutbound("http://127.0.0.1:1234"),
			},
		},
	}
	dispatcher := NewDispatcher(config)

	dispatcherStatus := dispatcher.Introspect()

	assert.Equal(t, config.Name, dispatcherStatus.Name)
	assert.NotEmpty(t, dispatcherStatus.ID)
	assert.Empty(t, dispatcherStatus.Procedures)
	assert.Len(t, dispatcherStatus.Inbounds, 1)
	assert.Len(t, dispatcherStatus.Outbounds, 2)

	inboundStatus := dispatcherStatus.Inbounds[0]
	assert.Equal(t, "http", inboundStatus.Transport)
	assert.Equal(t, "Stopped", inboundStatus.State)
	for _, outboundStatus := range dispatcherStatus.Outbounds {
		assert.Equal(t, "http", outboundStatus.Transport)
		assert.True(t, outboundStatus.RPCType == "unary" || outboundStatus.RPCType == "oneway")
		assert.Equal(t, "http://127.0.0.1:1234", outboundStatus.Endpoint)
		assert.Equal(t, "Stopped", outboundStatus.State)
		assert.Equal(t, "test-client", outboundStatus.Service)
		assert.Equal(t, "test-client", outboundStatus.OutboundKey)
	}

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

func checkPackageVersion(t *testing.T, packageNameToVersion map[string]string, key string, expectedVersion string) {
	version := packageNameToVersion[key]
	assert.NotEmpty(t, version)
	assert.Equal(t, expectedVersion, version)
}
