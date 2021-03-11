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

package yarpcconfig

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/internal/interpolate"
	"go.uber.org/yarpc/internal/whitespace"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
)

func TestConfiguratorRegisterErrors(t *testing.T) {
	require.Panics(t, func() { New().MustRegisterTransport(TransportSpec{}) })
	err := New().RegisterTransport(TransportSpec{})
	require.Error(t, err, "expected failure")
	assert.Contains(t, err.Error(), "name is required")
	err = New().RegisterTransport(TransportSpec{Name: "test"})
	require.Error(t, err, "expected failure")
	assert.Contains(t, err.Error(), "invalid TransportSpec for \"test\":")

	require.Panics(t, func() { New().MustRegisterPeerChooser(PeerChooserSpec{}) })
	err = New().RegisterPeerChooser(PeerChooserSpec{})
	require.Error(t, err, "expected failure")
	assert.Contains(t, err.Error(), "name is required")
	err = New().RegisterPeerChooser(PeerChooserSpec{Name: "test"})
	require.Error(t, err, "expected failure")
	assert.Contains(t, err.Error(), "invalid PeerChooserSpec for \"test\":")

	require.Panics(t, func() { New().MustRegisterPeerList(PeerListSpec{}) })
	err = New().RegisterPeerList(PeerListSpec{})
	require.Error(t, err, "expected failure")
	assert.Contains(t, err.Error(), "name is required")
	err = New().RegisterPeerList(PeerListSpec{Name: "test"})
	require.Error(t, err, "expected failure")
	assert.Contains(t, err.Error(), "invalid PeerListSpec for \"test\":")

	require.Panics(t, func() { New().MustRegisterPeerListUpdater(PeerListUpdaterSpec{}) })
	err = New().RegisterPeerListUpdater(PeerListUpdaterSpec{})
	require.Error(t, err, "expected failure")
	assert.Contains(t, err.Error(), "name is required")
	err = New().RegisterPeerListUpdater(PeerListUpdaterSpec{Name: "test"})
	require.Error(t, err, "expected failure")
	assert.Contains(t, err.Error(), "invalid PeerListUpdaterSpec for \"test\":")
}

func TestConfigurator(t *testing.T) {
	// For better test output, we have split the test case into a testCase
	// struct that defines the test parameters and a different anonymous
	// struct used in the table test to give a name to the test.

	type testCase struct {
		// List of TransportSpecs to register with the Configurator
		specs []TransportSpec

		// Name of the service or empty string to use the default
		serviceName string

		// YAML to parse using the configurator
		give string

		// Environment variables
		env map[string]string

		// If non-empty, an error is expected where the message matches all
		// strings in this slice
		wantErr []string

		// For success cases, the output Config must match this
		wantConfig yarpc.Config
	}

	tests := []struct {
		desc string
		test func(*testing.T, *gomock.Controller) testCase
	}{
		{
			desc: "no inbounds or outbounds",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.serviceName = "foo"
				tt.give = whitespace.Expand(``)
				tt.wantConfig = yarpc.Config{Name: "foo"}
				return
			},
		},
		{
			desc: "application error debug logging",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				debugLevel := zapcore.DebugLevel

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					logging:
						levels:
							applicationError: debug
				`)
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Logging: yarpc.LoggingConfig{
						Levels: yarpc.LogLevelConfig{
							ApplicationError: &debugLevel,
						},
					},
				}
				return
			},
		},
		{
			desc: "server error debug logging",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				debugLevel := zapcore.DebugLevel

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					logging:
						levels:
							serverError: debug
				`)
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Logging: yarpc.LoggingConfig{
						Levels: yarpc.LogLevelConfig{
							ServerError: &debugLevel,
						},
					},
				}
				return
			},
		},
		{
			desc: "client error debug logging",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				debugLevel := zapcore.DebugLevel

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					logging:
						levels:
							clientError: debug
				`)
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Logging: yarpc.LoggingConfig{
						Levels: yarpc.LogLevelConfig{
							ClientError: &debugLevel,
						},
					},
				}
				return
			},
		},
		{
			desc: "client and server error debug logging",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				debugLevel := zapcore.DebugLevel
				infoLevel := zapcore.InfoLevel

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					logging:
						levels:
							serverError: debug
							clientError: info
				`)
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Logging: yarpc.LoggingConfig{
						Levels: yarpc.LogLevelConfig{
							ClientError: &infoLevel,
							ServerError: &debugLevel,
						},
					},
				}
				return
			},
		},
		{
			desc: "outbound success info logging",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				debugLevel := zapcore.DebugLevel
				infoLevel := zapcore.InfoLevel

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					logging:
						levels:
							success: debug
							outbound:
								success: info
				`)
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Logging: yarpc.LoggingConfig{
						Levels: yarpc.LogLevelConfig{
							Success: &debugLevel,
							Outbound: yarpc.DirectionalLogLevelConfig{
								Success: &infoLevel,
							},
						},
					},
				}
				return
			},
		},
		{
			desc: "outbound success server error info logging",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				debugLevel := zapcore.DebugLevel
				infoLevel := zapcore.InfoLevel

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					logging:
						levels:
							serverError: debug
							outbound:
								serverError: info
				`)
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Logging: yarpc.LoggingConfig{
						Levels: yarpc.LogLevelConfig{
							ServerError: &debugLevel,
							Outbound: yarpc.DirectionalLogLevelConfig{
								ServerError: &infoLevel,
							},
						},
					},
				}
				return
			},
		},
		{
			desc: "outbound success client error info logging",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				debugLevel := zapcore.DebugLevel
				infoLevel := zapcore.InfoLevel

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					logging:
						levels:
							clientError: debug
							outbound:
								clientError: info
				`)
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Logging: yarpc.LoggingConfig{
						Levels: yarpc.LogLevelConfig{
							ClientError: &debugLevel,
							Outbound: yarpc.DirectionalLogLevelConfig{
								ClientError: &infoLevel,
							},
						},
					},
				}
				return
			},
		},
		{
			desc: "inbound success server error info logging",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				debugLevel := zapcore.DebugLevel
				infoLevel := zapcore.InfoLevel

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					logging:
						levels:
							serverError: debug
							inbound:
								serverError: info
				`)
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Logging: yarpc.LoggingConfig{
						Levels: yarpc.LogLevelConfig{
							ServerError: &debugLevel,
							Inbound: yarpc.DirectionalLogLevelConfig{
								ServerError: &infoLevel,
							},
						},
					},
				}
				return
			},
		},
		{
			desc: "inbound success client error info logging",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				debugLevel := zapcore.DebugLevel
				infoLevel := zapcore.InfoLevel

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					logging:
						levels:
							clientError: debug
							inbound:
								clientError: info
				`)
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Logging: yarpc.LoggingConfig{
						Levels: yarpc.LogLevelConfig{
							ClientError: &debugLevel,
							Inbound: yarpc.DirectionalLogLevelConfig{
								ClientError: &infoLevel,
							},
						},
					},
				}
				return
			},
		},
		{
			desc: "metric tags blocklist",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					metrics:
						tagsBlocklist:
							- "routing_delegate"
				`)
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Metrics: yarpc.MetricsConfig{
						TagsBlocklist: []string{
							"routing_delegate",
						},
					},
				}
				return
			},
		},
		{
			desc: "application error, invalid type",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
							applicationError: 42
				`)
				tt.wantErr = []string{
					"error decoding 'logging.levels.applicationError':",
					"could not decode Zap log level:",
					"expected type 'string', got unconvertible type 'int'",
				}
				return
			},
		},
		{
			desc: "application error, invalid level",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
							applicationError: not a level
				`)
				tt.wantErr = []string{
					"error decoding 'logging.levels.applicationError':",
					"could not decode Zap log level:",
					`unrecognized level: "not a level"`,
				}
				return
			},
		},
		{
			desc: "server error, invalid type",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
							serverError: 42
				`)
				tt.wantErr = []string{
					"error decoding 'logging.levels.serverError':",
					"could not decode Zap log level:",
					"expected type 'string', got unconvertible type 'int'",
				}
				return
			},
		},
		{
			desc: "server error, invalid level",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
							serverError: not a level
				`)
				tt.wantErr = []string{
					"error decoding 'logging.levels.serverError':",
					"could not decode Zap log level:",
					`unrecognized level: "not a level"`,
				}
				return
			},
		},
		{
			desc: "client error, invalid type",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
							clientError: 42
				`)
				tt.wantErr = []string{
					"error decoding 'logging.levels.clientError':",
					"could not decode Zap log level:",
					"expected type 'string', got unconvertible type 'int'",
				}
				return
			},
		},
		{
			desc: "client error, invalid level",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
							clientError: not a level
				`)
				tt.wantErr = []string{
					"error decoding 'logging.levels.clientError':",
					"could not decode Zap log level:",
					`unrecognized level: "not a level"`,
				}
				return
			},
		},
		{
			desc: "invalid usage of application error with server error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
							applicationError: debug
							serverError: debug
				`)
				tt.wantErr = []string{
					"invalid logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},
		{
			desc: "invalid usage of outbound application error with client error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  outbound:
								applicationError: debug
								clientError: debug
				`)
				tt.wantErr = []string{
					"invalid outbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},
		{
			desc: "invalid usage of outbound failure with server error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  outbound:
							  failure: debug
							  serverError: debug
				`)
				tt.wantErr = []string{
					"invalid outbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},
		{
			desc: "invalid usage of outbound failure with client error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  outbound:
							  failure: debug
							  clientError: debug
				`)
				tt.wantErr = []string{
					"invalid outbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},

		{
			desc: "invalid usage of outbound application error with server error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  outbound:
							  applicationError: debug
							  serverError: debug
				`)
				tt.wantErr = []string{
					"invalid outbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},
		{
			desc: "invalid usage of outbound application error with client error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  outbound:
							  applicationError: debug
							  clientError: debug
				`)
				tt.wantErr = []string{
					"invalid outbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},
		{
			desc: "invalid usage of outbound failure with server error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  outbound:
							  failure: debug
							  serverError: debug
				`)
				tt.wantErr = []string{
					"invalid outbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},
		{
			desc: "invalid usage of outbound failure with client error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  outbound:
							  failure: debug
							  clientError: debug
				`)
				tt.wantErr = []string{
					"invalid outbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},

		{
			desc: "invalid usage of inbound application error with client error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  inbound:
								applicationError: debug
								clientError: debug
				`)
				tt.wantErr = []string{
					"invalid inbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},
		{
			desc: "invalid usage of inbound failure with server error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  inbound:
							  failure: debug
							  serverError: debug
				`)
				tt.wantErr = []string{
					"invalid inbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},
		{
			desc: "invalid usage of inbound failure with client error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  inbound:
							  failure: debug
							  clientError: debug
				`)
				tt.wantErr = []string{
					"invalid inbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},

		{
			desc: "invalid usage of inbound application error with server error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  inbound:
							  applicationError: debug
							  serverError: debug
				`)
				tt.wantErr = []string{
					"invalid inbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},
		{
			desc: "invalid usage of inbound application error with client error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  inbound:
							  applicationError: debug
							  clientError: debug
				`)
				tt.wantErr = []string{
					"invalid inbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},
		{
			desc: "invalid usage of inbound failure with server error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  inbound:
							  failure: debug
							  serverError: debug
				`)
				tt.wantErr = []string{
					"invalid inbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},
		{
			desc: "invalid usage of inbound failure with client error",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					logging:
						levels:
						  inbound:
							  failure: debug
							  clientError: debug
				`)
				tt.wantErr = []string{
					"invalid inbound logging configuration, failure/applicationError configuration can not be used with serverError/clientError",
				}
				return
			},
		},
		{
			desc: "unknown inbound",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					inbounds:
						bar: {}
				`)
				tt.wantErr = []string{
					"failed to load inbound",
					`unknown transport "bar"`,
				}
				return
			},
		},
		{
			desc: "unknown implicit outbound",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					outbounds:
						myservice:
							http: {url: "http://localhost:8080/yarpc"}
				`)
				tt.wantErr = []string{
					`failed to load configuration for outbound "myservice"`,
					`unknown transport "http"`,
				}
				return
			},
		},
		{
			desc: "unknown unary outbound",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					outbounds:
						someservice:
							unary:
								tchannel:
									address: localhost:4040
				`)
				tt.wantErr = []string{
					`failed to load configuration for outbound "someservice"`,
					`unknown transport "tchannel"`,
				}
				return
			},
		},
		{
			desc: "unknown oneway outbound",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					outbounds:
						keyvalue:
							oneway:
								kafka: {queue: requests}
				`)
				tt.wantErr = []string{
					`failed to load configuration for outbound "keyvalue"`,
					`unknown transport "kafka"`,
				}
				return
			},
		},
		{
			desc: "unknown stream outbound",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					outbounds:
						keyvalue:
							stream:
								kafka: {queue: requests}
				`)
				tt.wantErr = []string{
					`failed to load configuration for outbound "keyvalue"`,
					`unknown transport "kafka"`,
				}
				return
			},
		},
		{
			desc: "unused transport",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type fooTransportConfig struct{ Items []int }

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					transports:
						bar:
							items: [1, 2, 3]
				`)

				foo := mockTransportSpecBuilder{
					Name:            "bar",
					TransportConfig: reflect.TypeOf(&fooTransportConfig{}),
				}.Build(mockCtrl)

				tt.specs = []TransportSpec{foo.Spec()}
				tt.wantConfig = yarpc.Config{Name: "foo"}

				return
			},
		},
		{
			desc: "transport config error",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type transportConfig struct{ KeepAlive time.Duration }
				type inboundConfig struct{ Address string }

				tt.give = whitespace.Expand(`
					inbounds:
						http: {address: ":80"}
					transports:
						http:
							keepAlive: thirty
				`)

				http := mockTransportSpecBuilder{
					Name:            "http",
					TransportConfig: reflect.TypeOf(&transportConfig{}),
					InboundConfig:   reflect.TypeOf(&inboundConfig{}),
				}.Build(mockCtrl)
				tt.specs = []TransportSpec{http.Spec()}

				tt.wantErr = []string{
					"failed to decode transport configuration:",
					"error decoding 'KeepAlive'",
					`invalid duration`,
					`thirty`,
				}

				return
			},
		},
		{
			desc: "inbound",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type inboundConfig struct{ Address string }
				tt.serviceName = "myservice"
				tt.give = whitespace.Expand(`
					inbounds:
						http: {address: ":80"}
				`)

				http := mockTransportSpecBuilder{
					Name:            "http",
					TransportConfig: _typeOfEmptyStruct,
					InboundConfig:   reflect.TypeOf(&inboundConfig{}),
				}.Build(mockCtrl)

				transport := transporttest.NewMockTransport(mockCtrl)
				inbound := transporttest.NewMockInbound(mockCtrl)

				http.EXPECT().
					BuildTransport(struct{}{}, kitMatcher{ServiceName: "myservice"}).
					Return(transport, nil)
				http.EXPECT().
					BuildInbound(
						&inboundConfig{Address: ":80"}, transport,
						kitMatcher{ServiceName: "myservice"}).
					Return(inbound, nil)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantConfig = yarpc.Config{
					Name:     "myservice",
					Inbounds: yarpc.Inbounds{inbound},
				}
				return
			},
		},
		{
			desc: "inbounds unsupported",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					inbounds:
						outgoing-only:
							foo: bar
				`)

				spec := mockTransportSpecBuilder{
					Name:            "outgoing-only",
					TransportConfig: _typeOfEmptyStruct,
				}.Build(mockCtrl)
				tt.specs = []TransportSpec{spec.Spec()}
				tt.wantErr = []string{
					`transport "outgoing-only" does not support inbound requests`,
				}

				return
			},
		},
		{
			desc: "duplicate inbounds",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type inboundConfig struct{ Address string }
				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					inbounds:
						http:
							address: ":8080"
						http2:
							type: http
							address: ":8081"
				`)

				http := mockTransportSpecBuilder{
					Name:            "http",
					TransportConfig: _typeOfEmptyStruct,
					InboundConfig:   reflect.TypeOf(&inboundConfig{}),
				}.Build(mockCtrl)
				transport := transporttest.NewMockTransport(mockCtrl)
				http.EXPECT().
					BuildTransport(struct{}{}, kitMatcher{ServiceName: "foo"}).
					Return(transport, nil)

				inbound := transporttest.NewMockInbound(mockCtrl)
				inbound2 := transporttest.NewMockInbound(mockCtrl)

				http.EXPECT().
					BuildInbound(
						&inboundConfig{Address: ":8080"},
						transport,
						kitMatcher{ServiceName: "foo"}).
					Return(inbound, nil)
				http.EXPECT().
					BuildInbound(
						&inboundConfig{Address: ":8081"},
						transport,
						kitMatcher{ServiceName: "foo"}).
					Return(inbound2, nil)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantConfig = yarpc.Config{
					Name:     "foo",
					Inbounds: yarpc.Inbounds{inbound, inbound2},
				}

				return
			},
		},
		{
			desc: "disabled inbound",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type inboundConfig struct{ Address string }
				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					inbounds:
						http:
							disabled: true
							address: ":8080"
						http2:
							type: http
							address: ":8081"
				`)

				http := mockTransportSpecBuilder{
					Name:            "http",
					TransportConfig: _typeOfEmptyStruct,
					InboundConfig:   reflect.TypeOf(&inboundConfig{}),
				}.Build(mockCtrl)

				transport := transporttest.NewMockTransport(mockCtrl)
				inbound := transporttest.NewMockInbound(mockCtrl)

				http.EXPECT().
					BuildTransport(struct{}{}, kitMatcher{ServiceName: "foo"}).
					Return(transport, nil)
				http.EXPECT().
					BuildInbound(
						&inboundConfig{Address: ":8081"},
						transport,
						kitMatcher{ServiceName: "foo"}).
					Return(inbound, nil)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantConfig = yarpc.Config{
					Name:     "foo",
					Inbounds: yarpc.Inbounds{inbound},
				}

				return
			},
		},
		{
			desc: "inbound error",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					inbounds:
						foo:
							unexpected: bar
				`)

				foo := mockTransportSpecBuilder{
					Name:            "foo",
					TransportConfig: _typeOfEmptyStruct,
					InboundConfig:   _typeOfEmptyStruct,
				}.Build(mockCtrl)
				tt.specs = []TransportSpec{foo.Spec()}
				tt.wantErr = []string{
					"failed to decode inbound configuration: failed to decode struct",
					"invalid keys: unexpected",
				}

				return
			},
		},
		{
			desc: "implicit outbound no support",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				tt.give = whitespace.Expand(`
					outbounds:
						myservice:
							sink:
								foo: bar
				`)

				sink := mockTransportSpecBuilder{
					Name:            "sink",
					TransportConfig: _typeOfEmptyStruct,
					InboundConfig:   _typeOfEmptyStruct,
				}.Build(mockCtrl)

				tt.specs = []TransportSpec{sink.Spec()}
				tt.wantErr = []string{`transport "sink" does not support outbound requests`}
				return
			},
		},
		{
			desc: "implicit outbound unary",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type outboundConfig struct{ Address string }
				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					outbounds:
						bar:
							tchannel:
								address: localhost:4040
				`)

				tchan := mockTransportSpecBuilder{
					Name:                "tchannel",
					TransportConfig:     _typeOfEmptyStruct,
					UnaryOutboundConfig: reflect.TypeOf(&outboundConfig{}),
				}.Build(mockCtrl)

				transport := transporttest.NewMockTransport(mockCtrl)
				outbound := transporttest.NewMockUnaryOutbound(mockCtrl)

				tchan.EXPECT().
					BuildTransport(struct{}{}, kitMatcher{ServiceName: "foo"}).
					Return(transport, nil)
				tchan.EXPECT().
					BuildUnaryOutbound(
						&outboundConfig{Address: "localhost:4040"},
						transport,
						kitMatcher{ServiceName: "foo"}).
					Return(outbound, nil)

				tt.specs = []TransportSpec{tchan.Spec()}
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Outbounds: yarpc.Outbounds{
						"bar": {
							Unary: outbound,
						},
					},
				}

				return
			},
		},
		{
			desc: "implicit outbound oneway",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type transportConfig struct{ Address string }
				type outboundConfig struct{ Queue string }
				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					outbounds:
						bar:
							redis:
								queue: requests
					transports:
						redis:
							address: localhost:6379
				`)

				redis := mockTransportSpecBuilder{
					Name:                 "redis",
					TransportConfig:      reflect.TypeOf(transportConfig{}),
					OnewayOutboundConfig: reflect.TypeOf(&outboundConfig{}),
				}.Build(mockCtrl)

				transport := transporttest.NewMockTransport(mockCtrl)
				outbound := transporttest.NewMockOnewayOutbound(mockCtrl)

				redis.EXPECT().
					BuildTransport(
						transportConfig{Address: "localhost:6379"},
						kitMatcher{ServiceName: "foo"}).
					Return(transport, nil)
				redis.EXPECT().
					BuildOnewayOutbound(
						&outboundConfig{Queue: "requests"},
						transport,
						kitMatcher{ServiceName: "foo"}).
					Return(outbound, nil)

				tt.specs = []TransportSpec{redis.Spec()}
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Outbounds: yarpc.Outbounds{
						"bar": {
							Oneway: outbound,
						},
					},
				}

				return
			},
		},
		{
			desc: "implicit outbound stream",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type transportConfig struct{ Address string }
				type outboundConfig struct{ Queue string }
				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					outbounds:
						bar:
							fake-stream-transport:
								queue: requests
					transports:
						fake-stream-transport:
							address: localhost:6379
				`)

				redis := mockTransportSpecBuilder{
					Name:                 "fake-stream-transport",
					TransportConfig:      reflect.TypeOf(transportConfig{}),
					StreamOutboundConfig: reflect.TypeOf(&outboundConfig{}),
				}.Build(mockCtrl)

				transport := transporttest.NewMockTransport(mockCtrl)
				outbound := transporttest.NewMockStreamOutbound(mockCtrl)

				redis.EXPECT().
					BuildTransport(
						transportConfig{Address: "localhost:6379"},
						kitMatcher{ServiceName: "foo"}).
					Return(transport, nil)
				redis.EXPECT().
					BuildStreamOutbound(
						&outboundConfig{Queue: "requests"},
						transport,
						kitMatcher{ServiceName: "foo"}).
					Return(outbound, nil)

				tt.specs = []TransportSpec{redis.Spec()}
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Outbounds: yarpc.Outbounds{
						"bar": {
							Stream: outbound,
						},
					},
				}

				return
			},
		},
		{
			desc: "implicit outbound unary, oneway, stream",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type transportConfig struct{ KeepAlive time.Duration }
				type outboundConfig struct{ URL string }
				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					outbounds:
						baz:
							http:
								url: http://localhost:8080/yarpc
					transports:
						http:
							keepAlive: 60s
				`)

				http := mockTransportSpecBuilder{
					Name:                 "http",
					TransportConfig:      reflect.TypeOf(&transportConfig{}),
					OnewayOutboundConfig: reflect.TypeOf(&outboundConfig{}),
					StreamOutboundConfig: reflect.TypeOf(&outboundConfig{}),
					UnaryOutboundConfig:  reflect.TypeOf(&outboundConfig{}),
				}.Build(mockCtrl)

				transport := transporttest.NewMockTransport(mockCtrl)
				unary := transporttest.NewMockUnaryOutbound(mockCtrl)
				oneway := transporttest.NewMockOnewayOutbound(mockCtrl)
				stream := transporttest.NewMockStreamOutbound(mockCtrl)

				http.EXPECT().
					BuildTransport(
						&transportConfig{KeepAlive: time.Minute},
						kitMatcher{ServiceName: "foo"}).
					Return(transport, nil)

				outcfg := outboundConfig{URL: "http://localhost:8080/yarpc"}
				http.EXPECT().
					BuildUnaryOutbound(&outcfg, transport, kitMatcher{ServiceName: "foo"}).
					Return(unary, nil)
				http.EXPECT().
					BuildOnewayOutbound(&outcfg, transport, kitMatcher{ServiceName: "foo"}).
					Return(oneway, nil)
				http.EXPECT().
					BuildStreamOutbound(&outcfg, transport, kitMatcher{ServiceName: "foo"}).
					Return(stream, nil)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Outbounds: yarpc.Outbounds{
						"baz": {
							Unary:  unary,
							Oneway: oneway,
							Stream: stream,
						},
					},
				}

				return
			},
		},
		{
			desc: "implicit outbound error",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type outboundConfig struct{ URL string }
				tt.give = whitespace.Expand(`
					outbounds:
						qux:
							http:
								uri: http://localhost:8080/yarpc
				`)

				http := mockTransportSpecBuilder{
					Name:                 "http",
					TransportConfig:      _typeOfEmptyStruct,
					OnewayOutboundConfig: reflect.TypeOf(&outboundConfig{}),
					UnaryOutboundConfig:  reflect.TypeOf(&outboundConfig{}),
					StreamOutboundConfig: reflect.TypeOf(&outboundConfig{}),
				}.Build(mockCtrl)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantErr = []string{
					`failed to add outbound "qux"`,
					"failed to decode oneway outbound configuration",
					"failed to decode unary outbound configuration",
					"failed to decode stream outbound configuration",
					"invalid keys: uri",
				}

				return
			},
		},
		{
			desc: "explicit outbounds",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type (
					httpOutboundConfig   struct{ URL string }
					httpTransportConfig  struct{ KeepAlive time.Duration }
					redisOutboundConfig  struct{ Queue string }
					redisTransportConfig struct{ Address string }
				)

				tt.serviceName = "myservice"
				tt.give = whitespace.Expand(`
					transports:
						http:
							keepAlive: 5m
						redis:
							address: "127.0.0.1:6379"
					outbounds:
						foo:
							unary:
								http:
									url: http://localhost:8080/yarpc/v1
							oneway:
								http:
									url: http://localhost:8081/yarpc/v2
							stream:
								http:
									url: http://localhost:8081/yarpc/v3
						bar:
							oneway:
								redis:
									queue: requests
				`)

				http := mockTransportSpecBuilder{
					Name:                 "http",
					TransportConfig:      reflect.TypeOf(httpTransportConfig{}),
					OnewayOutboundConfig: reflect.TypeOf(httpOutboundConfig{}),
					StreamOutboundConfig: reflect.TypeOf(httpOutboundConfig{}),
					UnaryOutboundConfig:  reflect.TypeOf(httpOutboundConfig{}),
				}.Build(mockCtrl)

				redis := mockTransportSpecBuilder{
					Name:                 "redis",
					TransportConfig:      reflect.TypeOf(redisTransportConfig{}),
					OnewayOutboundConfig: reflect.TypeOf(redisOutboundConfig{}),
				}.Build(mockCtrl)

				httpTransport := transporttest.NewMockTransport(mockCtrl)
				httpUnary := transporttest.NewMockUnaryOutbound(mockCtrl)
				httpOneway := transporttest.NewMockOnewayOutbound(mockCtrl)
				httpStream := transporttest.NewMockStreamOutbound(mockCtrl)
				http.EXPECT().
					BuildTransport(
						httpTransportConfig{KeepAlive: 5 * time.Minute},
						kitMatcher{ServiceName: "myservice"}).
					Return(httpTransport, nil)

				redisTransport := transporttest.NewMockTransport(mockCtrl)
				redisOneway := transporttest.NewMockOnewayOutbound(mockCtrl)
				redis.EXPECT().
					BuildTransport(
						redisTransportConfig{Address: "127.0.0.1:6379"},
						kitMatcher{ServiceName: "myservice"}).
					Return(redisTransport, nil)

				http.EXPECT().
					BuildUnaryOutbound(
						httpOutboundConfig{URL: "http://localhost:8080/yarpc/v1"},
						httpTransport,
						kitMatcher{ServiceName: "myservice"}).
					Return(httpUnary, nil)
				http.EXPECT().
					BuildOnewayOutbound(
						httpOutboundConfig{URL: "http://localhost:8081/yarpc/v2"},
						httpTransport,
						kitMatcher{ServiceName: "myservice"}).
					Return(httpOneway, nil)
				http.EXPECT().
					BuildStreamOutbound(
						httpOutboundConfig{URL: "http://localhost:8081/yarpc/v3"},
						httpTransport,
						kitMatcher{ServiceName: "myservice"}).
					Return(httpStream, nil)

				redis.EXPECT().
					BuildOnewayOutbound(
						redisOutboundConfig{Queue: "requests"},
						redisTransport,
						kitMatcher{ServiceName: "myservice"}).
					Return(redisOneway, nil)

				tt.specs = []TransportSpec{http.Spec(), redis.Spec()}
				tt.wantConfig = yarpc.Config{
					Name: "myservice",
					Outbounds: yarpc.Outbounds{
						"foo": {Unary: httpUnary, Oneway: httpOneway, Stream: httpStream},
						"bar": {Oneway: redisOneway},
					},
				}

				return
			},
		},
		{
			desc: "explicit unary error",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type outboundConfig struct{ URL string }
				tt.give = whitespace.Expand(`
					outbounds:
						hello:
							unary:
								http:
									scheme: https
									host: localhost
									port: 8088
									path: /yarpc
				`)

				http := mockTransportSpecBuilder{
					Name:                 "http",
					TransportConfig:      _typeOfEmptyStruct,
					OnewayOutboundConfig: reflect.TypeOf(&outboundConfig{}),
					UnaryOutboundConfig:  reflect.TypeOf(&outboundConfig{}),
				}.Build(mockCtrl)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantErr = []string{
					"failed to decode unary outbound configuration",
					"invalid keys: host, path, port, scheme",
				}

				return
			},
		},
		{
			desc: "explicit oneway error",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type outboundConfig struct{ Address string }
				tt.give = whitespace.Expand(`
					outbounds:
						hello:
							oneway:
								redis:
									host: localhost
									port: 6379
				`)

				redis := mockTransportSpecBuilder{
					Name:                 "redis",
					TransportConfig:      _typeOfEmptyStruct,
					OnewayOutboundConfig: reflect.TypeOf(&outboundConfig{}),
				}.Build(mockCtrl)

				tt.specs = []TransportSpec{redis.Spec()}
				tt.wantErr = []string{
					"failed to decode oneway outbound configuration",
					"invalid keys: host, port",
				}

				return
			},
		},
		{
			desc: "explicit stream error",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type outboundConfig struct{ Address string }
				tt.give = whitespace.Expand(`
					outbounds:
						hello:
							stream:
								redis:
									host: localhost
									port: 6379
				`)

				redis := mockTransportSpecBuilder{
					Name:                 "redis",
					TransportConfig:      _typeOfEmptyStruct,
					StreamOutboundConfig: reflect.TypeOf(&outboundConfig{}),
				}.Build(mockCtrl)

				tt.specs = []TransportSpec{redis.Spec()}
				tt.wantErr = []string{
					"failed to decode stream outbound configuration",
					"invalid keys: host, port",
				}

				return
			},
		},
		{
			desc: "explicit unary not supported",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type outboundConfig struct{ Queue string }
				tt.give = whitespace.Expand(`
					outbounds:
						bar:
							unary:
								redis:
									queue: requests
				`)

				redis := mockTransportSpecBuilder{
					Name:                 "redis",
					TransportConfig:      _typeOfEmptyStruct,
					OnewayOutboundConfig: reflect.TypeOf(&outboundConfig{}),
				}.Build(mockCtrl)

				tt.specs = []TransportSpec{redis.Spec()}
				tt.wantErr = []string{
					`transport "redis" does not support unary outbound requests`,
				}

				return
			},
		},
		{
			desc: "explicit oneway not supported",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type outboundConfig struct{ Address string }
				tt.give = whitespace.Expand(`
					outbounds:
						bar:
							oneway:
								tchannel:
									address: localhost:4040
				`)

				tchan := mockTransportSpecBuilder{
					Name:                "tchannel",
					TransportConfig:     _typeOfEmptyStruct,
					UnaryOutboundConfig: reflect.TypeOf(&outboundConfig{}),
				}.Build(mockCtrl)

				tt.specs = []TransportSpec{tchan.Spec()}
				tt.wantErr = []string{
					`transport "tchannel" does not support oneway outbound requests`,
				}

				return
			},
		},
		{
			desc: "explicit stream not supported",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type outboundConfig struct{ Address string }
				tt.give = whitespace.Expand(`
					outbounds:
						bar:
							stream:
								tchannel:
									address: localhost:4040
				`)

				tchan := mockTransportSpecBuilder{
					Name:                "tchannel",
					TransportConfig:     _typeOfEmptyStruct,
					UnaryOutboundConfig: reflect.TypeOf(&outboundConfig{}),
				}.Build(mockCtrl)

				tt.specs = []TransportSpec{tchan.Spec()}
				tt.wantErr = []string{
					`transport "tchannel" does not support stream outbound requests`,
				}

				return
			},
		},
		{
			desc: "implicit outbound service name override",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type outboundConfig struct{ URL string }
				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					outbounds:
						bar:
							http:
								url: http://localhost:8080/bar
						bar-staging:
							service: bar
							http:
								url: http://localhost:8081/bar
				`)

				http := mockTransportSpecBuilder{
					Name:                 "http",
					TransportConfig:      _typeOfEmptyStruct,
					UnaryOutboundConfig:  reflect.TypeOf(outboundConfig{}),
					OnewayOutboundConfig: reflect.TypeOf(outboundConfig{}),
				}.Build(mockCtrl)

				transport := transporttest.NewMockTransport(mockCtrl)
				unary := transporttest.NewMockUnaryOutbound(mockCtrl)
				oneway := transporttest.NewMockOnewayOutbound(mockCtrl)
				unaryStaging := transporttest.NewMockUnaryOutbound(mockCtrl)
				onewayStaging := transporttest.NewMockOnewayOutbound(mockCtrl)

				http.EXPECT().
					BuildTransport(struct{}{}, kitMatcher{ServiceName: "foo"}).
					Return(transport, nil)

				http.EXPECT().
					BuildUnaryOutbound(
						outboundConfig{URL: "http://localhost:8080/bar"},
						transport,
						kitMatcher{ServiceName: "foo"}).
					Return(unary, nil)
				http.EXPECT().
					BuildOnewayOutbound(
						outboundConfig{URL: "http://localhost:8080/bar"},
						transport,
						kitMatcher{ServiceName: "foo"}).
					Return(oneway, nil)

				http.EXPECT().
					BuildUnaryOutbound(
						outboundConfig{URL: "http://localhost:8081/bar"},
						transport,
						kitMatcher{ServiceName: "foo"}).
					Return(unaryStaging, nil)
				http.EXPECT().
					BuildOnewayOutbound(
						outboundConfig{URL: "http://localhost:8081/bar"},
						transport,
						kitMatcher{ServiceName: "foo"}).
					Return(onewayStaging, nil)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Outbounds: yarpc.Outbounds{
						"bar": {Unary: unary, Oneway: oneway},
						"bar-staging": {
							ServiceName: "bar",
							Unary:       unaryStaging,
							Oneway:      onewayStaging,
						},
					},
				}

				return
			},
		},
		{
			desc: "interpolated string",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type transportConfig struct {
					ServerAddress string `config:",interpolate"`
				}

				type outboundConfig struct {
					QueueName string `config:"queue,interpolate"`
				}

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					transports:
						redis:
							serverAddress: ${REDIS_ADDRESS}:${REDIS_PORT}
					outbounds:
						myservice:
							redis:
								queue: /${MYSERVICE_QUEUE}/inbound
				`)
				tt.env = map[string]string{
					"REDIS_ADDRESS":   "127.0.0.1",
					"REDIS_PORT":      "6379",
					"MYSERVICE_QUEUE": "myservice",
				}

				redis := mockTransportSpecBuilder{
					Name:                 "redis",
					TransportConfig:      reflect.TypeOf(transportConfig{}),
					OnewayOutboundConfig: reflect.TypeOf(outboundConfig{}),
				}.Build(mockCtrl)

				kit := kitMatcher{ServiceName: "foo"}
				transport := transporttest.NewMockTransport(mockCtrl)
				oneway := transporttest.NewMockOnewayOutbound(mockCtrl)

				redis.EXPECT().
					BuildTransport(transportConfig{ServerAddress: "127.0.0.1:6379"}, kit).
					Return(transport, nil)
				redis.EXPECT().
					BuildOnewayOutbound(outboundConfig{QueueName: "/myservice/inbound"}, transport, kit).
					Return(oneway, nil)

				tt.specs = []TransportSpec{redis.Spec()}
				tt.wantConfig = yarpc.Config{
					Name:      "foo",
					Outbounds: yarpc.Outbounds{"myservice": {Oneway: oneway}},
				}

				return
			},
		},
		{
			desc: "interpolated integer",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type inboundConfig struct {
					Port int `config:",interpolate"`
				}

				tt.serviceName = "hi"
				tt.give = whitespace.Expand(`
					inbounds:
						http:
							port: 1${HTTP_PORT}
				`)
				tt.env = map[string]string{"HTTP_PORT": "8080"}

				http := mockTransportSpecBuilder{
					Name:            "http",
					TransportConfig: _typeOfEmptyStruct,
					InboundConfig:   reflect.TypeOf(inboundConfig{}),
				}.Build(mockCtrl)

				kit := kitMatcher{ServiceName: "hi"}
				transport := transporttest.NewMockTransport(mockCtrl)
				inbound := transporttest.NewMockInbound(mockCtrl)

				http.EXPECT().BuildTransport(struct{}{}, kit).Return(transport, nil)
				http.EXPECT().
					BuildInbound(inboundConfig{Port: 18080}, transport, kit).
					Return(inbound, nil)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantConfig = yarpc.Config{
					Name:     "hi",
					Inbounds: yarpc.Inbounds{inbound},
				}

				return
			},
		},
		{
			desc: "intepolate non-string",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type inboundConfig struct {
					Port int `config:",interpolate"`
				}

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					inbounds:
						http:
							port: 80
				`)

				http := mockTransportSpecBuilder{
					Name:            "http",
					TransportConfig: _typeOfEmptyStruct,
					InboundConfig:   reflect.TypeOf(inboundConfig{}),
				}.Build(mockCtrl)

				kit := kitMatcher{ServiceName: "foo"}
				transport := transporttest.NewMockTransport(mockCtrl)
				inbound := transporttest.NewMockInbound(mockCtrl)

				http.EXPECT().BuildTransport(struct{}{}, kit).Return(transport, nil)
				http.EXPECT().
					BuildInbound(inboundConfig{Port: 80}, transport, kit).
					Return(inbound, nil)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantConfig = yarpc.Config{
					Name:     "foo",
					Inbounds: yarpc.Inbounds{inbound},
				}

				return
			},
		},
		{
			desc: "bad interpolation string",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type inboundConfig struct {
					Address string `config:",interpolate"`
				}

				tt.serviceName = "hi"
				tt.give = whitespace.Expand(`
					inbounds:
						http:
							address: :${HTTP_PORT
				`)
				tt.env = map[string]string{"HTTP_PORT": "8080"}

				http := mockTransportSpecBuilder{
					Name:            "http",
					TransportConfig: _typeOfEmptyStruct,
					InboundConfig:   reflect.TypeOf(inboundConfig{}),
				}.Build(mockCtrl)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantErr = []string{
					"failed to decode inbound configuration:",
					`error reading into field "Address":`,
					`failed to parse ":${HTTP_PORT" for interpolation`,
				}

				return
			},
		},
		{
			desc: "missing envvar",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type inboundConfig struct {
					Address string `config:",interpolate"`
				}

				tt.serviceName = "hi"
				tt.give = whitespace.Expand(`
					inbounds:
						http:
							address: :${HTTP_PORT}
				`)

				http := mockTransportSpecBuilder{
					Name:            "http",
					TransportConfig: _typeOfEmptyStruct,
					InboundConfig:   reflect.TypeOf(inboundConfig{}),
				}.Build(mockCtrl)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantErr = []string{
					"failed to decode inbound configuration:",
					`error reading into field "Address":`,
					`failed to render ":${HTTP_PORT}" with environment variables:`,
					`variable "HTTP_PORT" does not have a value or a default`,
				}

				return
			},
		},
		{
			desc: "time.Duration from env",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type inboundConfig struct {
					Timeout time.Duration `config:",interpolate"`
				}

				tt.serviceName = "foo"
				tt.give = whitespace.Expand(`
					inbounds:
						http:
							timeout: ${TIMEOUT}
				`)
				tt.env = map[string]string{"TIMEOUT": "5s"}

				http := mockTransportSpecBuilder{
					Name:            "http",
					TransportConfig: _typeOfEmptyStruct,
					InboundConfig:   reflect.TypeOf(inboundConfig{}),
				}.Build(mockCtrl)

				kit := kitMatcher{ServiceName: "foo"}
				transport := transporttest.NewMockTransport(mockCtrl)
				inbound := transporttest.NewMockInbound(mockCtrl)

				http.EXPECT().BuildTransport(struct{}{}, kit).Return(transport, nil)
				http.EXPECT().
					BuildInbound(inboundConfig{Timeout: 5 * time.Second}, transport, kit).
					Return(inbound, nil)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantConfig = yarpc.Config{
					Name:     "foo",
					Inbounds: yarpc.Inbounds{inbound},
				}

				return
			},
		},
	}

	// We want to parameterize all tests over YAML and non-YAML modes. To
	// avoid two layers of nesting, we let this helper function call our test
	// runner.
	runTest := func(name string, f func(t *testing.T, useYAML bool)) {
		t.Run(name, func(t *testing.T) {
			t.Run("yaml", func(t *testing.T) { f(t, true) })
			t.Run("direct", func(t *testing.T) { f(t, false) })
		})
	}

	for _, tc := range tests {
		runTest(tc.desc, func(t *testing.T, useYAML bool) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			tt := tc.test(t, mockCtrl)
			cfg := New(InterpolationResolver(mapVariableResolver(tt.env)))

			if tt.specs != nil {
				for _, spec := range tt.specs {
					err := cfg.RegisterTransport(spec)
					require.NoError(t, err, "failed to register transport %q", spec.Name)
				}
			}

			var (
				gotConfig yarpc.Config
				err       error
			)
			if useYAML {
				gotConfig, err = cfg.LoadConfigFromYAML(tt.serviceName, strings.NewReader(tt.give))
			} else {
				var data map[string]interface{}
				require.NoError(t, yaml.Unmarshal([]byte(tt.give), &data), "failed to parse YAML")

				gotConfig, err = cfg.LoadConfig(tt.serviceName, data)
			}

			if len(tt.wantErr) > 0 {
				require.Error(t, err, "expected failure")
				for _, msg := range tt.wantErr {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}

			require.NoError(t, err, "expected success")
			assert.Equal(t, tt.wantConfig, gotConfig, "config did not match")
		})
	}
}

func mapVariableResolver(m map[string]string) interpolate.VariableResolver {
	return func(name string) (value string, ok bool) {
		value, ok = m[name]
		return
	}
}
