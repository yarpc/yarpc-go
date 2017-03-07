package config

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestConfiguratorRegisterTransportMissingName(t *testing.T) {
	err := New().RegisterTransport(TransportSpec{})
	require.Error(t, err, "expected failure")
	assert.Contains(t, err.Error(), "name is required")
}

func TestConfigurator(t *testing.T) {
	// For better test output, we have split the test case into a testCase
	// struct that defines the test parameters and a different anonymous
	// struct used in the table test to give a name to the test.

	type testCase struct {
		// List of TransportSpecs to register with the Configurator
		specs []TransportSpec

		// YAML to parse using the configurator
		give string

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
			desc: "unknown inbound",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = untab(`
					name: foo
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
				tt.give = untab(`
					name: foo
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
				tt.give = untab(`
					name: foo
					outbounds:
					  someservice:
					    unary:
					      tchannel:
					        address: localhost:4040
				`)
				tt.wantErr = []string{
					`failed to load configuration for unary outbound "someservice"`,
					`unknown transport "tchannel"`,
				}
				return
			},
		},
		{
			desc: "unknown oneway outbound",
			test: func(*testing.T, *gomock.Controller) (tt testCase) {
				tt.give = untab(`
					name: foo
					outbounds:
					  keyvalue:
					    oneway:
					      redis: {queue: requests}
				`)
				tt.wantErr = []string{
					`failed to load configuration for oneway outbound "keyvalue"`,
					`unknown transport "redis"`,
				}
				return
			},
		},
		{
			desc: "unused transport",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type fooTransportConfig struct{ Items []int }

				tt.give = untab(`
					name: foo
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

				tt.give = untab(`
					name: foo
					inbounds:
					  http: {address: ":80"}
					transports:
					  http:
					    keepAlive: "thirty"
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
					"invalid duration thirty",
				}

				return
			},
		},
		{
			desc: "inbound",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type inboundConfig struct{ Address string }
				tt.give = untab(`
					name: foo
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
					BuildTransport(struct{}{}).
					Return(transport, nil)
				http.EXPECT().
					BuildInbound(&inboundConfig{Address: ":80"}, transport).
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
			desc: "inbounds unsupported",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				tt.give = untab(`
					name: foo
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
				tt.give = untab(`
					name: foo
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
				http.EXPECT().BuildTransport(struct{}{}).Return(transport, nil)

				inbound := transporttest.NewMockInbound(mockCtrl)
				inbound2 := transporttest.NewMockInbound(mockCtrl)

				http.EXPECT().
					BuildInbound(&inboundConfig{Address: ":8080"}, transport).
					Return(inbound, nil)
				http.EXPECT().
					BuildInbound(&inboundConfig{Address: ":8081"}, transport).
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
				tt.give = untab(`
					name: foo
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
					BuildTransport(struct{}{}).
					Return(transport, nil)
				http.EXPECT().
					BuildInbound(&inboundConfig{Address: ":8081"}, transport).
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
				tt.give = untab(`
					name: foo
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
			desc: "implicit outbound unary",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type outboundConfig struct{ Address string }
				tt.give = untab(`
					name: foo
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
					BuildTransport(struct{}{}).
					Return(transport, nil)
				tchan.EXPECT().
					BuildUnaryOutbound(&outboundConfig{Address: "localhost:4040"}, transport).
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
				tt.give = untab(`
					name: foo
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
					BuildTransport(transportConfig{Address: "localhost:6379"}).
					Return(transport, nil)
				redis.EXPECT().
					BuildOnewayOutbound(&outboundConfig{Queue: "requests"}, transport).
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
			desc: "implicit outbound unary and oneway",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type transportConfig struct{ KeepAlive time.Duration }
				type outboundConfig struct{ URL string }
				tt.give = untab(`
					name: foo
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
					UnaryOutboundConfig:  reflect.TypeOf(&outboundConfig{}),
				}.Build(mockCtrl)

				transport := transporttest.NewMockTransport(mockCtrl)
				unary := transporttest.NewMockUnaryOutbound(mockCtrl)
				oneway := transporttest.NewMockOnewayOutbound(mockCtrl)

				http.EXPECT().
					BuildTransport(&transportConfig{KeepAlive: time.Minute}).
					Return(transport, nil)

				outcfg := outboundConfig{URL: "http://localhost:8080/yarpc"}
				http.EXPECT().
					BuildUnaryOutbound(&outcfg, transport).
					Return(unary, nil)
				http.EXPECT().
					BuildOnewayOutbound(&outcfg, transport).
					Return(oneway, nil)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantConfig = yarpc.Config{
					Name: "foo",
					Outbounds: yarpc.Outbounds{
						"baz": {
							Unary:  unary,
							Oneway: oneway,
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
				tt.give = untab(`
					name: foo
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
				}.Build(mockCtrl)

				tt.specs = []TransportSpec{http.Spec()}
				tt.wantErr = []string{
					`failed to add outbound for "qux"`,
					"failed to decode oneway outbound configuration",
					"failed to decode unary outbound configuration",
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
				tt.give = untab(`
					name: myservice
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
					  bar:
					    oneway:
					      redis:
					        queue: requests
				`)

				http := mockTransportSpecBuilder{
					Name:                 "http",
					TransportConfig:      reflect.TypeOf(httpTransportConfig{}),
					OnewayOutboundConfig: reflect.TypeOf(httpOutboundConfig{}),
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
				http.EXPECT().
					BuildTransport(httpTransportConfig{KeepAlive: 5 * time.Minute}).
					Return(httpTransport, nil)

				redisTransport := transporttest.NewMockTransport(mockCtrl)
				redisOneway := transporttest.NewMockOnewayOutbound(mockCtrl)
				redis.EXPECT().
					BuildTransport(redisTransportConfig{Address: "127.0.0.1:6379"}).
					Return(redisTransport, nil)

				http.EXPECT().
					BuildUnaryOutbound(httpOutboundConfig{URL: "http://localhost:8080/yarpc/v1"}, httpTransport).
					Return(httpUnary, nil)
				http.EXPECT().
					BuildOnewayOutbound(httpOutboundConfig{URL: "http://localhost:8081/yarpc/v2"}, httpTransport).
					Return(httpOneway, nil)

				redis.EXPECT().
					BuildOnewayOutbound(redisOutboundConfig{Queue: "requests"}, redisTransport).
					Return(redisOneway, nil)

				tt.specs = []TransportSpec{http.Spec(), redis.Spec()}
				tt.wantConfig = yarpc.Config{
					Name: "myservice",
					Outbounds: yarpc.Outbounds{
						"foo": {Unary: httpUnary, Oneway: httpOneway},
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
				tt.give = untab(`
					name: foo
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
				tt.give = untab(`
					name: foo
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
			desc: "explicit unary not supported",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type outboundConfig struct{ Queue string }
				tt.give = untab(`
					name: foo
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
				tt.give = untab(`
					name: foo
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
			desc: "implicit outbound service name override",
			test: func(t *testing.T, mockCtrl *gomock.Controller) (tt testCase) {
				type outboundConfig struct{ URL string }
				tt.give = untab(`
					name: foo
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
					BuildTransport(struct{}{}).
					Return(transport, nil)

				http.EXPECT().
					BuildUnaryOutbound(outboundConfig{URL: "http://localhost:8080/bar"}, transport).
					Return(unary, nil)
				http.EXPECT().
					BuildOnewayOutbound(outboundConfig{URL: "http://localhost:8080/bar"}, transport).
					Return(oneway, nil)

				http.EXPECT().
					BuildUnaryOutbound(outboundConfig{URL: "http://localhost:8081/bar"}, transport).
					Return(unaryStaging, nil)
				http.EXPECT().
					BuildOnewayOutbound(outboundConfig{URL: "http://localhost:8081/bar"}, transport).
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
			cfg := New()

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
				gotConfig, err = cfg.LoadConfigFromYAML(strings.NewReader(tt.give))
			} else {
				var data map[string]interface{}
				require.NoError(t, yaml.Unmarshal([]byte(tt.give), &data), "failed to parse YAML")

				gotConfig, err = cfg.LoadConfig(data)
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

// Drop tabs from the start of every line of the given string.
func untab(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if len(l) == 0 {
			continue
		}

		lines[i] = strings.TrimLeft(l, "\t")
	}
	return strings.Join(lines, "\n")
}
