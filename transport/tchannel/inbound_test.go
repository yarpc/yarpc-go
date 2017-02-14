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
	"go.uber.org/yarpc/encoding/raw"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInboundStartNew(t *testing.T) {
	x, err := NewTransport(ServiceName("foo"))
	require.NoError(t, err)

	i := x.NewInbound()
	i.SetRouter(yarpc.NewMapRouter("foo"))
	require.NoError(t, i.Start())
	require.NoError(t, x.Start())
	require.NoError(t, i.Stop())
	require.NoError(t, x.Stop())
}

func TestInboundStopWithoutStarting(t *testing.T) {
	x, err := NewTransport(ServiceName("foo"))
	require.NoError(t, err)
	i := x.NewInbound()
	assert.NoError(t, i.Stop())
}

func TestInboundInvalidAddress(t *testing.T) {
	x, err := NewTransport(ServiceName("foo"), ListenAddr("not valid"))
	require.NoError(t, err)

	i := x.NewInbound()
	i.SetRouter(yarpc.NewMapRouter("foo"))
	assert.Nil(t, i.Start())
	defer i.Stop()
	assert.Error(t, x.Start())
	defer x.Stop()
}

type nophandler struct{}

func (nophandler) Handle(ctx context.Context, req *transport.Request,
	resw transport.ResponseWriter) error {
	resw.Write([]byte(req.Service))
	return nil
}

func TestInboundSubServices(t *testing.T) {
	itransport, err := NewTransport(ServiceName("myservice"), ListenAddr("localhost:0"))
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

	chservEndpoint := itransport.ch.PeerInfo().HostPort

	otransport, err := NewTransport(ServiceName("caller"))
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
