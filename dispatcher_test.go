// Copyright (c) 2016 Uber Technologies, Inc.
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
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/http"
	tch "go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/transport/transporttest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc"
)

func basicDispatcher(t *testing.T) Dispatcher {
	ch, err := tchannel.NewChannel("test", nil)
	require.NoError(t, err, "failed to create TChannel")

	return NewDispatcher(Config{
		Name: "test",
		Inbounds: []transport.Inbound{
			tch.NewInbound(ch, tch.ListenAddr(":0")),
			http.NewInbound(":0"),
		},
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
	assert.Implements(t,
		(*tch.Inbound)(nil), dispatcher.Inbounds()[0], "first inbound must be TChannel")
	assert.Implements(t,
		(*http.Inbound)(nil), dispatcher.Inbounds()[1], "second inbound must be HTTP")
}

func TestInboundsOrderAfterStart(t *testing.T) {
	dispatcher := basicDispatcher(t)

	require.NoError(t, dispatcher.Start(), "failed to start Dispatcher")
	defer dispatcher.Stop()

	inbounds := dispatcher.Inbounds()

	tchInbound := inbounds[0].(tch.Inbound)
	assert.NotEqual(t, "0.0.0.0:0", tchInbound.Channel().PeerInfo().HostPort)

	httpInbound := inbounds[1].(http.Inbound)
	assert.NotNil(t, httpInbound.Addr(), "expected an HTTP addr")
}

func TestStartStopFailures(t *testing.T) {
	tests := []struct {
		desc string

		inbounds  func(*gomock.Controller) []transport.Inbound
		outbounds func(*gomock.Controller) yarpc.Outbounds

		wantStartErr string
		wantStopErr  string
	}{
		{
			desc: "all success",
			inbounds: func(mockCtrl *gomock.Controller) []transport.Inbound {
				inbounds := make([]transport.Inbound, 10)
				for i := range inbounds {
					in := transporttest.NewMockInbound(mockCtrl)
					in.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil)
					in.EXPECT().Stop().Return(nil)
					inbounds[i] = in
				}
				return inbounds
			},
			outbounds: func(mockCtrl *gomock.Controller) yarpc.Outbounds {
				outbounds := make(yarpc.Outbounds, 10)
				for i := 0; i < 10; i++ {
					out := transporttest.NewMockUnaryOutbound(mockCtrl)
					out.EXPECT().Start(gomock.Any()).Return(nil)
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
			inbounds: func(mockCtrl *gomock.Controller) []transport.Inbound {
				inbounds := make([]transport.Inbound, 10)
				for i := range inbounds {
					in := transporttest.NewMockInbound(mockCtrl)
					if i == 6 {
						in.EXPECT().Start(gomock.Any(), gomock.Any()).Return(errors.New("great sadness"))
					} else {
						in.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil)
						in.EXPECT().Stop().Return(nil)
					}
					inbounds[i] = in
				}
				return inbounds
			},
			outbounds: func(mockCtrl *gomock.Controller) yarpc.Outbounds {
				outbounds := make(yarpc.Outbounds, 10)
				for i := 0; i < 10; i++ {
					out := transporttest.NewMockUnaryOutbound(mockCtrl)
					out.EXPECT().Start(gomock.Any()).Return(nil)
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
			inbounds: func(mockCtrl *gomock.Controller) []transport.Inbound {
				inbounds := make([]transport.Inbound, 10)
				for i := range inbounds {
					in := transporttest.NewMockInbound(mockCtrl)
					in.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil)
					if i == 7 {
						in.EXPECT().Stop().Return(errors.New("great sadness"))
					} else {
						in.EXPECT().Stop().Return(nil)
					}
					inbounds[i] = in
				}
				return inbounds
			},
			outbounds: func(mockCtrl *gomock.Controller) yarpc.Outbounds {
				outbounds := make(yarpc.Outbounds, 10)
				for i := 0; i < 10; i++ {
					out := transporttest.NewMockUnaryOutbound(mockCtrl)
					out.EXPECT().Start(gomock.Any()).Return(nil)
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
			inbounds: func(mockCtrl *gomock.Controller) []transport.Inbound {
				inbounds := make([]transport.Inbound, 10)
				for i := range inbounds {
					in := transporttest.NewMockInbound(mockCtrl)
					in.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil)
					in.EXPECT().Stop().Return(nil)
					inbounds[i] = in
				}
				return inbounds
			},
			outbounds: func(mockCtrl *gomock.Controller) yarpc.Outbounds {
				outbounds := make(yarpc.Outbounds, 10)
				for i := 0; i < 10; i++ {
					out := transporttest.NewMockUnaryOutbound(mockCtrl)
					if i == 5 {
						out.EXPECT().Start(gomock.Any()).Return(errors.New("something went wrong"))
					} else {
						out.EXPECT().Start(gomock.Any()).Return(nil)
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
			inbounds: func(mockCtrl *gomock.Controller) []transport.Inbound {
				inbounds := make([]transport.Inbound, 10)
				for i := range inbounds {
					in := transporttest.NewMockInbound(mockCtrl)
					in.EXPECT().Start(gomock.Any(), gomock.Any()).Return(nil)
					in.EXPECT().Stop().Return(nil)
					inbounds[i] = in
				}
				return inbounds
			},
			outbounds: func(mockCtrl *gomock.Controller) yarpc.Outbounds {
				outbounds := make(yarpc.Outbounds, 10)
				for i := 0; i < 10; i++ {
					out := transporttest.NewMockUnaryOutbound(mockCtrl)
					out.EXPECT().Start(gomock.Any()).Return(nil)
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

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	for _, tt := range tests {

		dispatcher := NewDispatcher(Config{
			Name:      "test",
			Inbounds:  tt.inbounds(mockCtrl),
			Outbounds: tt.outbounds(mockCtrl),
		})

		err := dispatcher.Start()
		if tt.wantStartErr != "" {
			if assert.Error(t, err, "%v: expected Start() to fail", tt.desc) {
				assert.Contains(t, err.Error(), tt.wantStartErr, tt.desc)
			}
			continue
		}
		if !assert.NoError(t, err, "%v: expected Start() to succeed", tt.desc) {
			continue
		}

		err = dispatcher.Stop()
		if tt.wantStopErr == "" {
			assert.NoError(t, err, "%v: expected Stop() to succeed", tt.desc)
			continue
		}
		if assert.Error(t, err, "%v: expected Stop() to fail", tt.desc) {
			assert.Contains(t, err.Error(), tt.wantStopErr, tt.desc)
		}
	}
}
