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
	"testing"

	. "go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func basicDispatcher(t *testing.T) *Dispatcher {
	httpTransport := http.NewTransport()
	tchannelTransport, err := tchannel.NewChannelTransport(tchannel.ServiceName("test"))
	require.NoError(t, err)

	return NewDispatcher(Config{
		Name: "test",
		Inbounds: Inbounds{
			tchannelTransport.NewInbound(),
			httpTransport.NewInbound(":0"),
		},
		Logger: zap.NewNop(),
	})
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
