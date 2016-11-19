package yarpc_test

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/raw"
	yhttp "go.uber.org/yarpc/transport/http"
	ytchannel "go.uber.org/yarpc/transport/tchannel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
	traw "github.com/uber/tchannel-go/raw"
	ncontext "golang.org/x/net/context"
)

var _reqBody = []byte("hello")

func yarpcEcho(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	return body, yarpc.NewResMeta().Headers(reqMeta.Headers()), nil
}

func httpEcho(t testing.TB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		hs := w.Header()
		for k, vs := range r.Header {
			hs[k] = vs
		}

		_, err := io.Copy(w, r.Body)
		assert.NoError(t, err, "failed to write HTTP response body")
	}
}

type tchannelEcho struct{ t testing.TB }

func (tchannelEcho) Handle(ctx ncontext.Context, args *traw.Args) (*traw.Res, error) {
	return &traw.Res{Arg2: args.Arg2, Arg3: args.Arg3}, nil
}

func (t tchannelEcho) OnError(ctx ncontext.Context, err error) {
	t.t.Fatalf("request failed: %v", err)
}

func withDispatcher(t testing.TB, cfg yarpc.Config, f func(yarpc.Dispatcher)) {
	d := yarpc.NewDispatcher(cfg)
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
		_, _, err := c.Call(ctx, yarpc.NewReqMeta().Procedure("echo"), _reqBody)
		require.NoError(b, err, "request %d failed", i+1)
	}
}

func runHTTPClient(b *testing.B, c *http.Client, url string) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		req, err := http.NewRequest("POST", url, bytes.NewReader(_reqBody))
		require.NoError(b, err, "failed to build request %d", i+1)
		req = req.WithContext(ctx)

		req.Header = http.Header{
			"Context-TTL-MS": {"100"},
			"Rpc-Caller":     {"http-client"},
			"Rpc-Encoding":   {"raw"},
			"Rpc-Procedure":  {"echo"},
			"Rpc-Service":    {"server"},
		}
		res, err := c.Do(req)
		require.NoError(b, err, "request %d failed", i+1)

		_, err = ioutil.ReadAll(res.Body)
		require.NoError(b, err, "failed to read response %d", i+1)
		require.NoError(b, res.Body.Close(), "failed to close response body %d", i+1)
	}
}

func runTChannelClient(b *testing.B, c *tchannel.Channel, hostPort string) {
	headers := []byte{0x00, 0x00} // TODO: YARPC TChannel should support empty arg2
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		call, err := c.BeginCall(ctx, hostPort, "server", "echo",
			&tchannel.CallOptions{Format: tchannel.Raw})
		require.NoError(b, err, "BeginCall %v failed", i+1)

		_, _, _, err = traw.WriteArgs(call, headers, _reqBody)
		require.NoError(b, err, "request %v failed", i+1)
	}
}

func Benchmark_HTTP_YARPCToYARPC(b *testing.B) {
	serverCfg := yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{yhttp.NewInbound(":8999")},
	}

	clientCfg := yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"server": {
				Unary: yhttp.NewOutbound("http://localhost:8999"),
			},
		},
	}

	withDispatcher(b, serverCfg, func(server yarpc.Dispatcher) {
		server.Register(raw.Procedure("echo", yarpcEcho))
		withDispatcher(b, clientCfg, func(client yarpc.Dispatcher) {
			b.ResetTimer()
			runYARPCClient(b, raw.New(client.Channel("server")))
		})
	})
}

func Benchmark_HTTP_YARPCToNetHTTP(b *testing.B) {
	clientCfg := yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"server": {
				Unary: yhttp.NewOutbound("http://localhost:8998"),
			},
		},
	}

	withHTTPServer(b, ":8998", httpEcho(b), func() {
		withDispatcher(b, clientCfg, func(client yarpc.Dispatcher) {
			b.ResetTimer()
			runYARPCClient(b, raw.New(client.Channel("server")))
		})
	})
}

func Benchmark_HTTP_NetHTTPToYARPC(b *testing.B) {
	serverCfg := yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{yhttp.NewInbound(":8996")},
	}

	withDispatcher(b, serverCfg, func(server yarpc.Dispatcher) {
		server.Register(raw.Procedure("echo", yarpcEcho))

		b.ResetTimer()
		runHTTPClient(b, http.DefaultClient, "http://localhost:8996")
	})
}

func Benchmark_HTTP_NetHTTPToNetHTTP(b *testing.B) {
	withHTTPServer(b, ":8997", httpEcho(b), func() {
		b.ResetTimer()
		runHTTPClient(b, http.DefaultClient, "http://localhost:8997")
	})
}

func Benchmark_TChannel_YARPCToYARPC(b *testing.B) {
	serverCh, err := tchannel.NewChannel("server", nil)
	require.NoError(b, err, "failed to build server TChannel")

	serverCfg := yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{ytchannel.NewInbound(serverCh)},
	}

	clientCh, err := tchannel.NewChannel("client", nil)
	require.NoError(b, err, "failed to build client TChannel")

	// no defer close on channels because YARPC will take care of that

	withDispatcher(b, serverCfg, func(server yarpc.Dispatcher) {
		server.Register(raw.Procedure("echo", yarpcEcho))

		// Need server already started to build client config
		clientCfg := yarpc.Config{
			Name: "client",
			Outbounds: yarpc.Outbounds{
				"server": {
					Unary: ytchannel.NewOutbound(clientCh, ytchannel.HostPort(serverCh.PeerInfo().HostPort)),
				},
			},
		}
		withDispatcher(b, clientCfg, func(client yarpc.Dispatcher) {
			b.ResetTimer()
			runYARPCClient(b, raw.New(client.Channel("server")))
		})
	})
}

func Benchmark_TChannel_YARPCToTChannel(b *testing.B) {
	serverCh, err := tchannel.NewChannel("server", nil)
	require.NoError(b, err, "failed to build server TChannel")
	defer serverCh.Close()

	serverCh.Register(traw.Wrap(tchannelEcho{t: b}), "echo")
	require.NoError(b, serverCh.ListenAndServe(":0"), "failed to start up TChannel")

	clientCh, err := tchannel.NewChannel("client", nil)
	require.NoError(b, err, "failed to build client TChannel")

	clientCfg := yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"server": {
				Unary: ytchannel.NewOutbound(clientCh, ytchannel.HostPort(serverCh.PeerInfo().HostPort)),
			},
		},
	}

	withDispatcher(b, clientCfg, func(client yarpc.Dispatcher) {
		b.ResetTimer()
		runYARPCClient(b, raw.New(client.Channel("server")))
	})
}

func Benchmark_TChannel_TChannelToYARPC(b *testing.B) {
	serverCh, err := tchannel.NewChannel("server", nil)
	require.NoError(b, err, "failed to build server TChannel")

	serverCfg := yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{ytchannel.NewInbound(serverCh)},
	}

	withDispatcher(b, serverCfg, func(server yarpc.Dispatcher) {
		server.Register(raw.Procedure("echo", yarpcEcho))

		clientCh, err := tchannel.NewChannel("client", nil)
		require.NoError(b, err, "failed to build client TChannel")
		defer clientCh.Close()

		b.ResetTimer()
		runTChannelClient(b, clientCh, serverCh.PeerInfo().HostPort)
	})
}

func Benchmark_TChannel_TChannelToTChannel(b *testing.B) {
	serverCh, err := tchannel.NewChannel("server", nil)
	require.NoError(b, err, "failed to build server TChannel")
	defer serverCh.Close()

	serverCh.Register(traw.Wrap(tchannelEcho{t: b}), "echo")
	require.NoError(b, serverCh.ListenAndServe(":0"), "failed to start up TChannel")

	clientCh, err := tchannel.NewChannel("client", nil)
	require.NoError(b, err, "failed to build client TChannel")
	defer clientCh.Close()

	b.ResetTimer()
	runTChannelClient(b, clientCh, serverCh.PeerInfo().HostPort)
}
