// Copyright (c) 2025 Uber Technologies, Inc.
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
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	traw "github.com/uber/tchannel-go/raw"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	yhttp "go.uber.org/yarpc/transport/http"
	ytchannel "go.uber.org/yarpc/transport/tchannel"
	ncontext "golang.org/x/net/context"
)

var _reqBody = []byte("hello")

func yarpcEcho(ctx context.Context, body []byte) ([]byte, error) {
	call := yarpc.CallFromContext(ctx)
	for _, k := range call.HeaderNames() {
		if err := call.WriteResponseHeader(k, call.Header(k)); err != nil {
			return nil, err
		}
	}
	return body, nil
}

func httpEcho(t testing.TB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		hs := w.Header()
		for k, vs := range r.Header {
			hs[k] = vs
		}

		_, err := io.Copy(w, r.Body)
		if err != nil {
			t.Errorf("failed to write HTTP response body: %v", err)
		}
	}
}

type tchannelEcho struct{ t testing.TB }

func (tchannelEcho) Handle(ctx ncontext.Context, args *traw.Args) (*traw.Res, error) {
	return &traw.Res{Arg2: args.Arg2, Arg3: args.Arg3}, nil
}

func (t tchannelEcho) OnError(ctx ncontext.Context, err error) {
	t.t.Fatalf("request failed: %v", err)
}

func withDispatcher(t testing.TB, cfg yarpc.Config, f func(*yarpc.Dispatcher), ps ...[]transport.Procedure) {
	d := yarpc.NewDispatcher(cfg)
	for _, p := range ps {
		d.Register(p)
	}
	require.NoError(t, d.Start(), "failed to start server")
	defer d.Stop()

	f(d)
}

func withHTTPServer(t testing.TB, listenOn string, h http.Handler, f func()) {
	l, err := net.Listen("tcp", listenOn)
	require.NoError(t, err, "could not listen on %q", listenOn)

	ch := make(chan struct{})
	go func() {
		http.Serve(l, h)
		close(ch)
	}()
	f()
	assert.NoError(t, l.Close(), "failed to stop listener on %q", listenOn)
	<-ch // wait until server has stopped
}

func runYARPCClient(b *testing.B, c raw.Client) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_, err := c.Call(ctx, "echo", _reqBody)
		if err != nil {
			b.Errorf("request %d failed: %v", i+1, err)
		}
	}
}

func runHTTPClient(b *testing.B, c *http.Client, url string) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		req, err := http.NewRequest("POST", url, bytes.NewReader(_reqBody))
		if err != nil {
			b.Errorf("failed to build request %d: %v", i+1, err)
		}
		req = req.WithContext(ctx)

		req.Header = http.Header{
			"Context-TTL-MS": {"100"},
			"Rpc-Caller":     {"http-client"},
			"Rpc-Encoding":   {"raw"},
			"Rpc-Procedure":  {"echo"},
			"Rpc-Service":    {"server"},
		}
		res, err := c.Do(req)
		if err != nil {
			b.Errorf("request %d failed: %v", i+1, err)
		}

		if _, err := io.ReadAll(res.Body); err != nil {
			b.Errorf("failed to read response %d: %v", i+1, err)
		}
		if err := res.Body.Close(); err != nil {
			b.Errorf("failed to close response body %d: %v", i+1, err)
		}
	}
}

func runTChannelClient(b *testing.B, c *tchannel.Channel, hostPort string) {
	headers := []byte{0x00, 0x00} // TODO: YARPC TChannel should support empty arg2
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		call, err := c.BeginCall(ctx, hostPort, "server", "echo",
			&tchannel.CallOptions{Format: tchannel.Raw})

		if err != nil {
			b.Errorf("BeginCall %v failed: %v", i+1, err)
		}

		_, _, _, err = traw.WriteArgs(call, headers, _reqBody)
		if err != nil {
			b.Errorf("request %v failed: %v", i+1, err)
		}
	}
}

func Benchmark_HTTP_YARPCToYARPC(b *testing.B) {
	httpTransport := yhttp.NewTransport()
	serverCfg := yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{httpTransport.NewInbound("127.0.0.1:8999")},
	}

	clientCfg := yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"server": {
				Unary: httpTransport.NewSingleOutbound("http://localhost:8999"),
			},
		},
	}

	withDispatcher(
		b, serverCfg,
		func(server *yarpc.Dispatcher) {
			withDispatcher(b, clientCfg, func(client *yarpc.Dispatcher) {
				b.ResetTimer()
				runYARPCClient(b, raw.New(client.ClientConfig("server")))
			})
		},
		raw.Procedure("echo", yarpcEcho),
	)
}

func Benchmark_HTTP_YARPCToNetHTTP(b *testing.B) {
	httpTransport := yhttp.NewTransport()
	clientCfg := yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"server": {
				Unary: httpTransport.NewSingleOutbound("http://localhost:8998"),
			},
		},
	}

	withHTTPServer(b, ":8998", httpEcho(b), func() {
		withDispatcher(b, clientCfg, func(client *yarpc.Dispatcher) {
			b.ResetTimer()
			runYARPCClient(b, raw.New(client.ClientConfig("server")))
		})
	})
}

func Benchmark_HTTP_NetHTTPToYARPC(b *testing.B) {
	httpTransport := yhttp.NewTransport()
	serverCfg := yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{httpTransport.NewInbound("127.0.0.1:8996")},
	}

	withDispatcher(
		b, serverCfg, func(server *yarpc.Dispatcher) {
			b.ResetTimer()
			runHTTPClient(b, http.DefaultClient, "http://localhost:8996")
		},
		raw.Procedure("echo", yarpcEcho),
	)
}

func Benchmark_HTTP_NetHTTPToNetHTTP(b *testing.B) {
	withHTTPServer(b, ":8997", httpEcho(b), func() {
		b.ResetTimer()
		runHTTPClient(b, http.DefaultClient, "http://localhost:8997")
	})
}

func Benchmark_TChannel_YARPCToYARPC(b *testing.B) {
	serverTChannel, err := ytchannel.NewChannelTransport(ytchannel.ServiceName("server"))
	require.NoError(b, err)

	serverCfg := yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{serverTChannel.NewInbound()},
	}

	clientTChannel, err := ytchannel.NewChannelTransport(ytchannel.ServiceName("client"))
	require.NoError(b, err)

	// no defer close on channels because YARPC will take care of that

	withDispatcher(
		b, serverCfg, func(server *yarpc.Dispatcher) {
			// Need server already started to build client config
			clientCfg := yarpc.Config{
				Name: "client",
				Outbounds: yarpc.Outbounds{
					"server": {
						Unary: clientTChannel.NewSingleOutbound(serverTChannel.ListenAddr()),
					},
				},
			}
			withDispatcher(b, clientCfg, func(client *yarpc.Dispatcher) {
				b.ResetTimer()
				runYARPCClient(b, raw.New(client.ClientConfig("server")))
			})
		},
		raw.Procedure("echo", yarpcEcho),
	)
}

func Benchmark_TChannel_YARPCToTChannel(b *testing.B) {
	serverCh, err := tchannel.NewChannel("server", nil)
	require.NoError(b, err, "failed to build server TChannel")
	defer serverCh.Close()

	serverCh.Register(traw.Wrap(tchannelEcho{t: b}), "echo")
	require.NoError(b, serverCh.ListenAndServe("127.0.0.1:0"), "failed to start up TChannel")

	clientTChannel, err := ytchannel.NewChannelTransport(ytchannel.ServiceName("client"))
	require.NoError(b, err)

	clientCfg := yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"server": {
				Unary: clientTChannel.NewSingleOutbound(serverCh.PeerInfo().HostPort),
			},
		},
	}

	withDispatcher(b, clientCfg, func(client *yarpc.Dispatcher) {
		b.ResetTimer()
		runYARPCClient(b, raw.New(client.ClientConfig("server")))
	})
}

func Benchmark_TChannel_TChannelToYARPC(b *testing.B) {
	tchannelTransport, err := ytchannel.NewChannelTransport(ytchannel.ServiceName("server"))
	require.NoError(b, err)

	serverCfg := yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{tchannelTransport.NewInbound()},
	}

	withDispatcher(
		b, serverCfg, func(dispatcher *yarpc.Dispatcher) {

			clientCh, err := tchannel.NewChannel("client", nil)
			require.NoError(b, err, "failed to build client TChannel")
			defer clientCh.Close()

			b.ResetTimer()
			runTChannelClient(b, clientCh, tchannelTransport.ListenAddr())
		},
		raw.Procedure("echo", yarpcEcho),
	)
}

func Benchmark_TChannel_TChannelToTChannel(b *testing.B) {
	serverCh, err := tchannel.NewChannel("server", nil)
	require.NoError(b, err, "failed to build server TChannel")
	defer serverCh.Close()

	serverCh.Register(traw.Wrap(tchannelEcho{t: b}), "echo")
	require.NoError(b, serverCh.ListenAndServe("127.0.0.1:0"), "failed to start up TChannel")

	clientCh, err := tchannel.NewChannel("client", nil)
	require.NoError(b, err, "failed to build client TChannel")
	defer clientCh.Close()

	b.ResetTimer()
	runTChannelClient(b, clientCh, serverCh.PeerInfo().HostPort)
}

func BenchmarkHTTPRoundTripper(b *testing.B) {
	uri := "http://localhost:8001"

	outbound := yhttp.NewTransport().NewSingleOutbound(uri)
	require.NoError(b, outbound.Start())
	defer outbound.Stop()

	roundTripper := &http.Client{Transport: outbound}

	withHTTPServer(b, ":8001", httpEcho(b), func() {
		b.ResetTimer()
		runHTTPClient(b, roundTripper, uri)
	})
}
