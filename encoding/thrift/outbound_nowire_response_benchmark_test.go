// Copyright (c) 2026 Uber Technologies, Inc.
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

package thrift

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/pkg/procedure"
	yhttp "go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
)

// BenchmarkNoWireResponseDecode uses many short Thrift strings rather than one
// large binary field: each string requires a length read and a payload read.
// The TChannel cases control for responses already buffered by the transport.
func BenchmarkNoWireResponseDecode(b *testing.B) {
	for _, transportName := range []string{"http2", "tchannel", "tchannel_pooled"} {
		b.Run(transportName, func(b *testing.B) {
			for _, shape := range []struct {
				name  string
				count int
			}{
				{name: "small", count: 8},
				{name: "large", count: 1280},
			} {
				b.Run(shape.name, func(b *testing.B) {
					values := make([]string, shape.count)
					for i := range values {
						values[i] = strings.Repeat("x", 96)
					}
					payload := encodeStringList(b, values)
					client := newResponseBenchmarkClient(b, transportName, payload)
					b.ReportAllocs()
					b.SetBytes(int64(len(payload)))
					b.ResetTimer()
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
							var result stringListResponse
							err := client.Call(ctx, fakeEnveloper(wire.Call), &result)
							cancel()
							if err != nil {
								b.Error(err)
								return
							}
							if len(result) != shape.count || result[len(result)-1] != values[len(values)-1] {
								b.Error("incorrect response")
								return
							}
						}
					})
				})
			}
		})
	}
}

type responseBenchmarkHandler struct {
	payload []byte
}

func (h responseBenchmarkHandler) Handle(_ context.Context, req *transport.Request, rw transport.ResponseWriter) error {
	if _, err := io.Copy(io.Discard, req.Body); err != nil {
		return err
	}
	_, err := rw.Write(h.payload)
	return err
}

func newResponseBenchmarkClient(b *testing.B, transportName string, payload []byte) NoWireClient {
	b.Helper()
	const (
		caller = "caller"
		callee = "service"
	)
	var (
		inbound  transport.Inbound
		outbound transport.UnaryOutbound
		starters []transport.Lifecycle
		httpIn   *yhttp.Inbound
	)
	if transportName == "http2" {
		serverTransport := yhttp.NewTransport()
		httpIn = serverTransport.NewInbound("127.0.0.1:0")
		inbound = httpIn
	} else {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(b, err)
		serverTransport, err := tchannel.NewTransport(tchannel.ServiceName(callee), tchannel.Listener(listener))
		require.NoError(b, err)
		inbound = serverTransport.NewInbound()
		clientTransport, err := tchannel.NewTransport(tchannel.ServiceName(caller))
		require.NoError(b, err)
		out := clientTransport.NewSingleOutbound(listener.Addr().String(),
			tchannel.WithReuseBuffer(transportName == "tchannel_pooled"))
		outbound = out
		starters = append(starters, clientTransport, out)
	}
	dispatcher := yarpc.NewDispatcher(yarpc.Config{Name: callee, Inbounds: yarpc.Inbounds{inbound}})
	dispatcher.Register([]transport.Procedure{{
		Name:        procedure.ToName("MyService", "someMethod"),
		Service:     callee,
		Encoding:    Encoding,
		HandlerSpec: transport.NewUnaryHandlerSpec(responseBenchmarkHandler{payload: payload}),
	}})
	require.NoError(b, dispatcher.Start())
	b.Cleanup(func() { require.NoError(b, dispatcher.Stop()) })
	if httpIn != nil {
		clientTransport := yhttp.NewTransport()
		out := clientTransport.NewSingleOutbound("http://"+httpIn.Addr().String(), yhttp.UseHTTP2())
		outbound = out
		starters = append(starters, clientTransport, out)
	}
	for _, starter := range starters {
		require.NoError(b, starter.Start())
		b.Cleanup(func() { require.NoError(b, starter.Stop()) })
	}
	return NewNoWire(Config{
		Service:      "MyService",
		ClientConfig: clientconfig.MultiOutbound(caller, callee, transport.Outbounds{Unary: outbound}),
	})
}
