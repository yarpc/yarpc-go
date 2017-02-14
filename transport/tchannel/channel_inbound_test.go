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

package tchannel

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/transport/transporttest"
	"go.uber.org/yarpc/encoding/raw"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/json"
	"github.com/uber/tchannel-go/testutils"
)

func TestChannelInboundStartNew(t *testing.T) {
	ch, err := tchannel.NewChannel("foo", nil)
	require.NoError(t, err)

	x, err := NewChannelTransport(WithChannel(ch))
	require.NoError(t, err)

	i := x.NewInbound()
	mctrl := gomock.NewController(t)
	router := transporttest.NewMockRouter(mctrl)
	router.EXPECT().Procedures().Return(nil)
	i.SetRouter(router)
	// Can't do Equal because we want to match the pointer, not a
	// DeepEqual.
	assert.True(t, ch == i.Channel(), "channel does not match")
	require.NoError(t, i.Start())
	require.NoError(t, x.Start())

	assert.Equal(t, tchannel.ChannelListening, ch.State())
	assert.NoError(t, i.Stop())
	assert.NoError(t, x.Stop())
	assert.Equal(t, tchannel.ChannelClosed, ch.State())
}

func TestChannelInboundStartAlreadyListening(t *testing.T) {
	ch, err := tchannel.NewChannel("foo", nil)
	require.NoError(t, err)

	require.NoError(t, ch.ListenAndServe(":0"))
	assert.Equal(t, tchannel.ChannelListening, ch.State())

	x, err := NewChannelTransport(WithChannel(ch))
	require.NoError(t, err)

	i := x.NewInbound()

	mctrl := gomock.NewController(t)
	router := transporttest.NewMockRouter(mctrl)
	router.EXPECT().Procedures().Return(nil)
	i.SetRouter(router)
	require.NoError(t, i.Start())
	require.NoError(t, x.Start())
	assert.Equal(t, tchannel.ChannelListening, ch.State())

	assert.NoError(t, i.Stop())
	assert.NoError(t, x.Stop())
	assert.Equal(t, tchannel.ChannelClosed, ch.State())
}

func TestChannelInboundStopWithoutStarting(t *testing.T) {
	ch, err := tchannel.NewChannel("foo", nil)
	require.NoError(t, err)

	x, err := NewChannelTransport(WithChannel(ch))
	require.NoError(t, err)

	i := x.NewInbound()
	assert.NoError(t, i.Stop())
}

func TestChannelInboundInvalidAddress(t *testing.T) {
	x, err := NewChannelTransport(ServiceName("foo"), ListenAddr("not valid"))
	require.NoError(t, err)

	i := x.NewInbound()
	mctrl := gomock.NewController(t)
	router := transporttest.NewMockRouter(mctrl)
	router.EXPECT().Procedures().Return(nil)
	i.SetRouter(router)
	assert.Nil(t, i.Start())
	defer i.Stop()
	assert.Error(t, x.Start())
	defer x.Stop()
}

func TestChannelInboundExistingMethods(t *testing.T) {
	// Create a channel with an existing "echo" method.
	ch, err := tchannel.NewChannel("foo", nil)
	require.NoError(t, err)
	json.Register(ch, json.Handlers{
		"echo": func(ctx json.Context, req map[string]string) (map[string]string, error) {
			return req, nil
		},
	}, nil)

	x, err := NewChannelTransport(WithChannel(ch))
	require.NoError(t, err)

	i := x.NewInbound()
	mctrl := gomock.NewController(t)
	router := transporttest.NewMockRouter(mctrl)
	router.EXPECT().Procedures().Return(nil)
	i.SetRouter(router)
	require.NoError(t, i.Start())
	defer i.Stop()
	require.NoError(t, x.Start())
	defer x.Stop()

	// Make a call to the "echo" method which should call our pre-registered method.
	ctx, cancel := json.NewContext(time.Second)
	defer cancel()

	var resp map[string]string
	arg := map[string]string{"k": "v"}

	svc := ch.ServiceName()
	peer := ch.Peers().GetOrAdd(ch.PeerInfo().HostPort)
	err = json.CallPeer(ctx, peer, svc, "echo", arg, &resp)
	require.NoError(t, err, "Call failed")
	assert.Equal(t, arg, resp, "Response mismatch")
}

func TestChannelInboundSubServices(t *testing.T) {
	chserv := testutils.NewServer(t, nil)
	defer chserv.Close()
	chservEndpoint := chserv.PeerInfo().HostPort

	itransport, err := NewChannelTransport(ServiceName("myservice"), WithChannel(chserv))
	require.NoError(t, err)

	router := yarpc.NewMapRouter("myservice")

	i := itransport.NewInbound()
	i.SetRouter(router)

	nophandlerspec := transport.NewUnaryHandlerSpec(nophandler{})

	router.Register([]transport.Procedure{
		{Name: "hello", HandlerSpec: nophandlerspec},
		{Service: "subservice", Name: "hello", HandlerSpec: nophandlerspec},
		{Service: "subservice", Name: "world", HandlerSpec: nophandlerspec},
		{Service: "subservice2", Name: "hello", HandlerSpec: nophandlerspec},
		{Service: "subservice2", Name: "monde", HandlerSpec: nophandlerspec},
	})

	require.NoError(t, i.Start())
	require.NoError(t, itransport.Start())

	otransport, err := NewChannelTransport(ServiceName("caller"))
	require.NoError(t, err)
	o := otransport.NewSingleOutbound(chservEndpoint)

	require.NoError(t, o.Start())
	defer o.Stop()

	for _, tt := range []struct {
		service   string
		procedure string
	}{
		{"myservice", "hello"},
		{"subservice", "hello"},
		{"subservice", "world"},
		{"subservice2", "hello"},
		{"subservice2", "monde"},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		res, err := o.Call(
			ctx,
			&transport.Request{
				Caller:    "caller",
				Service:   tt.service,
				Procedure: tt.procedure,
				Encoding:  raw.Encoding,
				Body:      bytes.NewReader([]byte{}),
			},
		)
		if !assert.NoError(t, err, "failed to make call") {
			continue
		}
		if !assert.Equal(t, false, res.ApplicationError, "not application error") {
			continue
		}
		body, err := ioutil.ReadAll(res.Body)
		if !assert.NoError(t, err) {
			continue
		}
		assert.Equal(t, string(body), tt.service)
	}

	require.NoError(t, i.Stop())
	require.NoError(t, itransport.Stop())
}
