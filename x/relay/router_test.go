package relay

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
)

func TestRouting(t *testing.T) {
	type shardProxyDef struct {
		shard       string
		outboundkey string
		isUnary     bool
	}
	type serviceProxyDef struct {
		service     string
		outboundkey string
		isUnary     bool
	}
	type procedureProxyDef struct {
		service     string
		procedure   string
		outboundkey string
		isUnary     bool
	}
	testCases := []struct {
		name string

		// give{Unary,Oneway}Outbounds specifies a series of mock unary/oneway
		// outbounds to create in the dispatcher.
		giveUnaryOutbounds  []string
		giveOnewayOutbounds []string

		// giveOutboundResponseErr is the error returned from all outbounds.
		giveOutboundResponseErr error

		// giveShardProxies specifies proxy definitions for shardkey-level
		// handlers.
		giveShardProxies []shardProxyDef

		// giveServiceProxies specifies proxy definitions for service-level
		// handlers.
		giveServiceProxies []serviceProxyDef

		// giveProcedureProxies specifies proxy definitions for procedure-level
		// handlers.
		giveProcedureProxies []procedureProxyDef

		// giveRequest is the request used per test.
		giveRequest *transport.Request

		// expectToCallOutbound is the name of the outbound we expect to call
		// for the request.
		expectToCallOutbound string

		// expectChooseErr is the error we expect to be returned from the
		// router's `Choose` method.
		expectChooseErr error

		// expectHandlerErr is the error we expect to be returned from the
		// proxy handler.
		expectHandlerErr error

		// These counters are the tally-compatible "name+tags" that point to
		// the current counter value.
		wantCounters map[string]int
	}{
		{
			name:               "unary service proxy",
			giveUnaryOutbounds: []string{"testOut"},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "test",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
			},
			expectToCallOutbound: "testOut",
			giveServiceProxies: []serviceProxyDef{
				{service: "test", outboundkey: "testOut", isUnary: true},
			},
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                       1,
				"frontcar_choose_successes+match_type=service": 1,
			},
		},
		{
			name:               "unary service proxy error",
			giveUnaryOutbounds: []string{"testOut"},
			giveServiceProxies: []serviceProxyDef{
				{service: "test", outboundkey: "testOut", isUnary: true},
			},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "test",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
			},
			giveOutboundResponseErr: fmt.Errorf("Test error"),
			expectHandlerErr:        fmt.Errorf("Test error"),
			expectToCallOutbound:    "testOut",
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                       1,
				"frontcar_choose_successes+match_type=service": 1,
			},
		},
		{
			name:               "multiple unary service proxies",
			giveUnaryOutbounds: []string{"testOut1", "testOut2", "testOut3"},
			giveServiceProxies: []serviceProxyDef{
				{service: "testOut1", outboundkey: "testOut1", isUnary: true},
				{service: "testOut2", outboundkey: "testOut2", isUnary: true},
				{service: "testOut3", outboundkey: "testOut3", isUnary: true},
			},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "testOut2",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
			},
			expectToCallOutbound: "testOut2",
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                       1,
				"frontcar_choose_successes+match_type=service": 1,
			},
		},
		{
			name:                "simple oneway service proxy",
			giveOnewayOutbounds: []string{"testOut"},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "test",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
			},
			expectToCallOutbound: "testOut",
			giveServiceProxies: []serviceProxyDef{
				{service: "test", outboundkey: "testOut", isUnary: false},
			},
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                       1,
				"frontcar_choose_successes+match_type=service": 1,
			},
		},
		{
			name:                "oneway proxy service error",
			giveOnewayOutbounds: []string{"testOut"},
			giveServiceProxies: []serviceProxyDef{
				{service: "testOut", outboundkey: "testOut", isUnary: false},
			},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "testOut",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
			},
			giveOutboundResponseErr: fmt.Errorf("Test error"),
			expectHandlerErr:        fmt.Errorf("Test error"),
			expectToCallOutbound:    "testOut",
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                       1,
				"frontcar_choose_successes+match_type=service": 1,
			},
		},
		{
			name:                "multiple oneway giveServiceProxies",
			giveOnewayOutbounds: []string{"testOut1", "testOut2", "testOut3"},
			giveServiceProxies: []serviceProxyDef{
				{service: "testOut1", outboundkey: "testOut1", isUnary: false},
				{service: "testOut2", outboundkey: "testOut2", isUnary: false},
				{service: "testOut3", outboundkey: "testOut3", isUnary: false},
			},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "testOut2",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
			},
			expectToCallOutbound: "testOut2",
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                       1,
				"frontcar_choose_successes+match_type=service": 1,
			},
		},
		{
			name: "missing procedure",
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "invalidService",
				Procedure: "invalidProcedure",
				Encoding:  "doesnotmatter",
			},
			expectChooseErr: transport.UnrecognizedProcedureError(
				&transport.Request{
					Service:   "invalidService",
					Procedure: "invalidProcedure",
				},
			),
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                       1,
				"frontcar_choose_successes+match_type=service": 1,
			},
		},
		{
			name:               "prioritize routing to a dispatcher procedure",
			giveUnaryOutbounds: []string{"procedureOut", "serviceOut", "shardOut"},
			giveServiceProxies: []serviceProxyDef{
				{service: "myservice", outboundkey: "serviceOut", isUnary: true},
			},
			giveShardProxies: []shardProxyDef{
				{shard: "myshard", outboundkey: "shardOut", isUnary: true},
			},
			giveProcedureProxies: []procedureProxyDef{
				{service: "myservice", procedure: "myprocedure", outboundkey: "procedureOut", isUnary: true},
			},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "myservice",
				Procedure: "myprocedure",
				Encoding:  "doesnotmatter",
				ShardKey:  "myshard",
			},
			expectToCallOutbound: "procedureOut",
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                                 1,
				"frontcar_choose_successes+match_type=service_procedure": 1,
			},
		},
		{
			name:                "prioritize routing oneway to a dispatcher procedure",
			giveOnewayOutbounds: []string{"procedureOut", "serviceOut", "shardOut"},
			giveServiceProxies: []serviceProxyDef{
				{service: "myservice", outboundkey: "serviceOut", isUnary: false},
			},
			giveShardProxies: []shardProxyDef{
				{shard: "myshard", outboundkey: "shardOut", isUnary: false},
			},
			giveProcedureProxies: []procedureProxyDef{
				{service: "myservice", procedure: "myprocedure", outboundkey: "procedureOut", isUnary: false},
			},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "myservice",
				Procedure: "myprocedure",
				Encoding:  "doesnotmatter",
				ShardKey:  "myshard",
			},
			expectToCallOutbound: "procedureOut",
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                                 1,
				"frontcar_choose_successes+match_type=service_procedure": 1,
			},
		},
		{
			name:               "unary shard proxy",
			giveUnaryOutbounds: []string{"testOut"},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "doesnotmatter",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
				ShardKey:  "test",
			},
			expectToCallOutbound: "testOut",
			giveShardProxies: []shardProxyDef{
				{shard: "test", outboundkey: "testOut", isUnary: true},
			},
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                         1,
				"frontcar_choose_successes+match_type=shard_key": 1,
			},
		},
		{
			name:               "unary shard proxy error",
			giveUnaryOutbounds: []string{"testOut"},
			giveShardProxies: []shardProxyDef{
				{shard: "test", outboundkey: "testOut", isUnary: true},
			},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "doesnotmatter",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
				ShardKey:  "test",
			},
			giveOutboundResponseErr: fmt.Errorf("Test error"),
			expectHandlerErr:        fmt.Errorf("Test error"),
			expectToCallOutbound:    "testOut",
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                         1,
				"frontcar_choose_successes+match_type=shard_key": 1,
			},
		},
		{
			name:               "multiple unary shard proxies",
			giveUnaryOutbounds: []string{"testOut1", "testOut2", "testOut3"},
			giveShardProxies: []shardProxyDef{
				{shard: "testOut1", outboundkey: "testOut1", isUnary: true},
				{shard: "testOut2", outboundkey: "testOut2", isUnary: true},
				{shard: "testOut3", outboundkey: "testOut3", isUnary: true},
			},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "doesnotmatter",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
				ShardKey:  "testOut2",
			},
			expectToCallOutbound: "testOut2",
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                         1,
				"frontcar_choose_successes+match_type=shard_key": 1,
			},
		},
		{
			name:                "simple oneway shard proxy",
			giveOnewayOutbounds: []string{"testOut"},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "doesnotmatter",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
				ShardKey:  "test",
			},
			expectToCallOutbound: "testOut",
			giveShardProxies: []shardProxyDef{
				{shard: "test", outboundkey: "testOut", isUnary: false},
			},
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                         1,
				"frontcar_choose_successes+match_type=shard_key": 1,
			},
		},
		{
			name:                "oneway proxy shard error",
			giveOnewayOutbounds: []string{"testOut"},
			giveShardProxies: []shardProxyDef{
				{shard: "testOut", outboundkey: "testOut", isUnary: false},
			},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "doesnotmatter",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
				ShardKey:  "testOut",
			},
			giveOutboundResponseErr: fmt.Errorf("Test error"),
			expectHandlerErr:        fmt.Errorf("Test error"),
			expectToCallOutbound:    "testOut",
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                         1,
				"frontcar_choose_successes+match_type=shard_key": 1,
			},
		},
		{
			name:                "multiple oneway shard proxies",
			giveOnewayOutbounds: []string{"testOut1", "testOut2", "testOut3"},
			giveShardProxies: []shardProxyDef{
				{shard: "testOut1", outboundkey: "testOut1", isUnary: false},
				{shard: "testOut2", outboundkey: "testOut2", isUnary: false},
				{shard: "testOut3", outboundkey: "testOut3", isUnary: false},
			},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "doesnotmatter",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
				ShardKey:  "testOut2",
			},
			expectToCallOutbound: "testOut2",
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                         1,
				"frontcar_choose_successes+match_type=shard_key": 1,
			},
		},
		{
			name:               "prioritize routing to a service procedure",
			giveUnaryOutbounds: []string{"serviceOut", "shardOut"},
			giveServiceProxies: []serviceProxyDef{
				{service: "myservice", outboundkey: "serviceOut", isUnary: true},
			},
			giveShardProxies: []shardProxyDef{
				{shard: "myshard", outboundkey: "shardOut", isUnary: true},
			},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "myservice",
				Procedure: "myprocedure",
				Encoding:  "doesnotmatter",
				ShardKey:  "myshard",
			},
			expectToCallOutbound: "serviceOut",
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                       1,
				"frontcar_choose_successes+match_type=service": 1,
			},
		},
		{
			name:                "prioritize routing oneway to a service procedure",
			giveOnewayOutbounds: []string{"serviceOut", "shardOut"},
			giveServiceProxies: []serviceProxyDef{
				{service: "myservice", outboundkey: "serviceOut", isUnary: false},
			},
			giveShardProxies: []shardProxyDef{
				{shard: "myshard", outboundkey: "shardOut", isUnary: false},
			},
			giveRequest: &transport.Request{
				Caller:    "doesnotmatter",
				Service:   "myservice",
				Procedure: "doesnotmatter",
				Encoding:  "doesnotmatter",
				ShardKey:  "myshard",
			},
			expectToCallOutbound: "serviceOut",
			wantCounters: map[string]int{
				"frontcar_choose_calls+":                       1,
				"frontcar_choose_successes+match_type=service": 1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			unaryResponse := []byte("This is a test response")

			outbounds := make(yarpc.Outbounds, len(tc.giveUnaryOutbounds)+len(tc.giveOnewayOutbounds))
			for _, outKey := range tc.giveUnaryOutbounds {
				out := transporttest.NewMockUnaryOutbound(mockCtrl)
				out.EXPECT().Transports().AnyTimes().Return([]transport.Transport{})
				out.EXPECT().Start().AnyTimes()
				out.EXPECT().Stop().AnyTimes()
				if tc.expectToCallOutbound == outKey {
					resp := &transport.Response{
						Body: ioutil.NopCloser(bytes.NewBuffer(unaryResponse)),
					}
					out.EXPECT().Call(ctx, tc.giveRequest).Return(resp, tc.giveOutboundResponseErr)
				}
				outbounds[outKey] = transport.Outbounds{
					Unary: out,
				}
			}
			for _, outKey := range tc.giveOnewayOutbounds {
				out := transporttest.NewMockOnewayOutbound(mockCtrl)
				out.EXPECT().Transports().AnyTimes().Return([]transport.Transport{})
				out.EXPECT().Start().AnyTimes()
				out.EXPECT().Stop().AnyTimes()
				if tc.expectToCallOutbound == outKey {
					out.EXPECT().CallOneway(ctx, tc.giveRequest).Return(time.Now(), tc.giveOutboundResponseErr)
				}
				outbounds[outKey] = transport.Outbounds{
					Oneway: out,
				}
			}

			cfg := yarpc.Config{
				Name:      "service",
				Outbounds: outbounds,
				Metrics: yarpc.MetricsConfig{
					Tally: tally.NoopScope,
				},
			}

			testScope := tally.NewTestScope("", map[string]string{})
			sr := NewRouter(Scope(testScope))
			cfg.RouterMiddleware = sr
			dispatcher := yarpc.NewDispatcher(cfg)

			numDefaultProcs := len(dispatcher.Router().Procedures())

			for _, proxy := range tc.giveShardProxies {
				handler := ShardKeyHandler{ShardKey: proxy.shard}
				if proxy.isUnary {
					handler.HandlerSpec = transport.NewUnaryHandlerSpec(
						UnaryProxyHandler(dispatcher.ClientConfig(proxy.outboundkey).GetUnaryOutbound()),
					)
				} else {
					handler.HandlerSpec = transport.NewOnewayHandlerSpec(
						OnewayProxyHandler(dispatcher.ClientConfig(proxy.outboundkey).GetOnewayOutbound()),
					)
				}
				sr.RegisterShard([]ShardKeyHandler{handler})
			}

			for _, proxy := range tc.giveServiceProxies {
				handler := ServiceHandler{Service: proxy.service}
				if proxy.isUnary {
					handler.HandlerSpec = transport.NewUnaryHandlerSpec(
						UnaryProxyHandler(dispatcher.ClientConfig(proxy.outboundkey).GetUnaryOutbound()),
					)
				} else {
					handler.HandlerSpec = transport.NewOnewayHandlerSpec(
						OnewayProxyHandler(dispatcher.ClientConfig(proxy.outboundkey).GetOnewayOutbound()),
					)
				}
				sr.RegisterService([]ServiceHandler{handler})
			}

			for _, proxy := range tc.giveProcedureProxies {
				proc := transport.Procedure{
					Name:    proxy.procedure,
					Service: proxy.service,
				}
				if proxy.isUnary {
					proc.HandlerSpec = transport.NewUnaryHandlerSpec(
						UnaryProxyHandler(dispatcher.ClientConfig(proxy.outboundkey).GetUnaryOutbound()),
					)
				} else {
					proc.HandlerSpec = transport.NewOnewayHandlerSpec(
						OnewayProxyHandler(dispatcher.ClientConfig(proxy.outboundkey).GetOnewayOutbound()),
					)
				}
				dispatcher.Register([]transport.Procedure{proc})
			}

			require.NoError(t, dispatcher.Start())
			defer dispatcher.Stop()

			// Validate that the number of procedures equals the procedure & service proxies + the default procedures
			assert.Len(t, dispatcher.Router().Procedures(), len(tc.giveShardProxies)+len(tc.giveProcedureProxies)+len(tc.giveServiceProxies)+numDefaultProcs)

			spec, err := dispatcher.Router().Choose(ctx, tc.giveRequest)
			if tc.expectChooseErr != nil {
				assert.EqualError(t, err, tc.expectChooseErr.Error())
				return
			}
			require.NoError(t, err)

			switch spec.Type() {
			case transport.Unary:
				respWriter := new(transporttest.FakeResponseWriter)
				err = spec.Unary().Handle(ctx, tc.giveRequest, respWriter)
				if err == nil {
					assert.Equal(t, unaryResponse, respWriter.Body.Bytes())
				}
			case transport.Oneway:
				err = spec.Oneway().HandleOneway(ctx, tc.giveRequest)
			default:
				panic("unexpected spec type")
			}
			if tc.expectHandlerErr != nil {
				assert.EqualError(t, err, tc.expectHandlerErr.Error())
			} else {
				assert.NoError(t, err)
			}

			counters := testScope.Snapshot().Counters()
			for nameAndTags, value := range tc.wantCounters {
				require.Contains(t, counters, nameAndTags, "name+tag combo was not in the counters")
				assert.Equal(t, int64(value), counters[nameAndTags].Value(), "counter %s was not as expected", nameAndTags)
			}
		})
	}
}
