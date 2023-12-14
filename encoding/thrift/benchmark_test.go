package thrift_test

import (
	"context"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/examples/thrift-keyvalue/keyvalue/kv"
	"go.uber.org/yarpc/internal/examples/thrift-keyvalue/keyvalue/kv/keyvalueclient"
	"go.uber.org/yarpc/internal/examples/thrift-keyvalue/keyvalue/kv/keyvalueserver"
	"go.uber.org/yarpc/transport/tchannel"
	"gonum.org/v1/gonum/stat/distuv"
)

const (
	_kvServer = "callee"
	_kvClient = "caller"
)

type generator struct {
	norm distuv.Normal
	min  int
	max  int
}

func (g generator) next() int {
	out := int(g.norm.Rand())
	if out < g.min {
		out = 0
	}
	if out > g.max {
		out = g.max
	}
	return out
}

func BenchmarkThriftClientCallNormalDist(b *testing.B) {
	handler := &keyValueHandler{}
	serverAddr := newKeyValServer(b, handler)

	clientNoReuse := newKeyValueClient(b, serverAddr, false)
	clientWithReuse := newKeyValueClient(b, serverAddr, true)

	// Create a normal distribution
	g := generator{
		norm: distuv.Normal{
			Mu:    3 * 1024, // 3KB
			Sigma: 10000,
		},
		min: 0,
		max: 2 * 1024 * 1024, // 2MB
	}

	var samples []string
	for i := 0; i < 10000; i++ {
		key := "foo" + strconv.FormatInt(int64(i), 10)
		len := g.next()
		value := generateRandomString(len)
		samples = append(samples, value)
		handler.SetValue(context.Background(), &key, &value)
	}

	b.ResetTimer()

	b.Run("with_buffer_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			offset := i % len(samples)
			key := "foo" + strconv.FormatInt(int64(offset), 10)
			value := samples[i%len(samples)]
			callGetter(b, clientWithReuse, key, value)
		}
	})

	b.Run("without_buffer_pool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			offset := i % len(samples)
			key := "foo" + strconv.FormatInt(int64(offset), 10)
			value := samples[i%len(samples)]
			callGetter(b, clientNoReuse, key, value)
		}
	})
}

func generateRandomString(len int) string {
	var sb strings.Builder
	for i := 0; i < len; i++ {
		c := 'a' + rand.Intn('z'-'a')
		sb.WriteByte(byte(c))
	}
	return sb.String()
}

func callGetter(b *testing.B, client keyvalueclient.Interface, key string, want string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got, err := client.GetValue(ctx, &key)
	require.NoError(b, err)
	require.Equal(b, want, got)
}

type keyValueHandler struct {
	items sync.Map
}

func (h *keyValueHandler) GetValue(ctx context.Context, key *string) (string, error) {
	if v, ok := h.items.Load(*key); ok {
		return v.(string), nil
	}
	return "", &kv.ResourceDoesNotExist{Key: *key}
}

func (h *keyValueHandler) SetValue(ctx context.Context, key *string, value *string) error {
	h.items.Store(*key, *value)
	return nil
}

func newKeyValServer(t testing.TB, handler keyvalueserver.Interface) string {
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	trans, err := tchannel.NewTransport(
		tchannel.ServiceName(_kvServer),
		tchannel.Listener(listen))
	require.NoError(t, err)

	inbound := trans.NewInbound()
	addr := listen.Addr().String()

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name:     _kvServer,
		Inbounds: yarpc.Inbounds{inbound},
	})

	dispatcher.Register(keyvalueserver.New(handler))
	require.NoError(t, dispatcher.Start(), "could not start server dispatcher")

	t.Cleanup(func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") })

	return addr
}

func newKeyValueClient(t testing.TB, serverAddr string, enableBufferReuse bool) keyvalueclient.Interface {
	trans, err := tchannel.NewTransport(tchannel.ServiceName(_kvClient))
	require.NoError(t, err)
	out := trans.NewSingleOutbound(serverAddr, tchannel.WithReuseBuffer(enableBufferReuse))

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: _kvClient,
		Outbounds: map[string]transport.Outbounds{
			_kvServer: {
				ServiceName: _kvServer,
				Unary:       out,
			},
		},
	})

	client := keyvalueclient.New(dispatcher.ClientConfig(_kvServer))
	require.NoError(t, dispatcher.Start(), "could not start client dispatcher")

	t.Cleanup(func() { assert.NoError(t, dispatcher.Stop(), "could not stop dispatcher") })
	return client
}
