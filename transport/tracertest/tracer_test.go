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

package tracertest

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
	ytchannel "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
)

type echoReqBody struct{}
type echoResBody struct{}

func echo(ctx context.Context, reqMeta yarpc.ReqMeta, reqBody *echoReqBody) (*echoResBody, yarpc.ResMeta, error) {
	return &echoResBody{}, nil, nil
}

func TestHttpInboundTracer(t *testing.T) {
	tracer := mocktracer.New()
	rpc := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: []transport.Inbound{
			http.NewInbound(":8080"),
		},
		Outbounds: transport.Outbounds{
			"yarpc-test": http.NewOutbound("http://localhost:8080"),
		},
		Tracer: tracer,
	})

	json.Register(rpc, json.Procedure("echo", echo))

	client := json.New(rpc.Channel("yarpc-test"))

	rpc.Start()
	defer rpc.Stop()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	var resBody echoResBody
	_, err := client.Call(
		ctx,
		yarpc.NewReqMeta().Procedure("echo"),
		&echoReqBody{},
		&resBody,
	)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(tracer.FinishedSpans()), "generates one span")
	assert.Equal(t, tracer.FinishedSpans()[0].OperationName, "echo", "span has correct operation name")
}

func TestTChannelInboundTracer(t *testing.T) {

	tracer := mocktracer.New()

	// Establish the TChannel
	ch, err := tchannel.NewChannel("yarpc-test", &tchannel.ChannelOptions{
		Tracer: tracer,
	})
	assert.NoError(t, err)
	ip, err := tchannel.ListenIP()
	assert.NoError(t, err)
	hp := fmt.Sprintf("%v:%v", ip, 4040)
	ch.ListenAndServe(hp)

	rpc := yarpc.NewDispatcher(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: []transport.Inbound{
			ytchannel.NewInbound(ch),
		},
		Outbounds: transport.Outbounds{
			"yarpc-test": ytchannel.NewOutbound(ch, ytchannel.HostPort(hp)),
		},
		Tracer: tracer,
	})

	json.Register(rpc, json.Procedure("echo", echo))

	client := json.New(rpc.Channel("yarpc-test"))

	rpc.Start()
	defer rpc.Stop()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	var resBody echoResBody
	_, err = client.Call(
		ctx,
		yarpc.NewReqMeta().Procedure("echo"),
		&echoReqBody{},
		&resBody,
	)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(tracer.FinishedSpans()), "generates one span")
	if len(tracer.FinishedSpans()) != 1 {
		return
	}
	assert.Equal(t, tracer.FinishedSpans()[0].OperationName, "echo", "span has correct operation name")
}
