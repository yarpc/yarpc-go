package recorder

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport"
	"go.uber.org/yarpc/transport/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestSanitizeFilename(t *testing.T) {
	assert.EqualValues(t, sanitizeFilename(`hello`), `hello`)
	assert.EqualValues(t, sanitizeFilename(`h/e\l?l%o*`), `h_e_l_l_o_`)
	assert.EqualValues(t, sanitizeFilename(`:h|e"l<l>o.`), `_h_e_l_l_o.`)
	assert.EqualValues(t, sanitizeFilename(`10€|çí¹`), `10__çí¹`)
	assert.EqualValues(t, sanitizeFilename("hel\x00lo"), `hel_lo`)
}

type randomGenerator struct {
	randsrc *rand.Rand
}

func newRandomGenerator(seed int64) randomGenerator {
	return randomGenerator{
		randsrc: rand.New(rand.NewSource(seed)),
	}
}

// Atom returns an ASCII string.
func (r *randomGenerator) Atom() string {
	length := 3 + r.randsrc.Intn(13)
	atom := make([]byte, length)
	for i := 0; i < length; i++ {
		letter := r.randsrc.Intn(2 * 26)
		if letter < 26 {
			atom[i] = 'A' + byte(letter)
		} else {
			atom[i] = 'a' + byte(letter-26)
		}
	}
	return string(atom)
}

// Headers returns a new randomized header.
func (r *randomGenerator) Headers() transport.Headers {
	headers := transport.NewHeaders()
	size := 2 + r.randsrc.Intn(6)
	for i := 0; i < size; i++ {
		headers = headers.With(r.Atom(), r.Atom())
	}
	return headers
}

// Request returns a new randomized request.
func (r *randomGenerator) Request() (transport.Request, []byte) {
	bodyData := []byte(r.Atom())

	return transport.Request{
		Caller:          r.Atom(),
		Service:         r.Atom(),
		Encoding:        transport.Encoding(r.Atom()),
		Procedure:       r.Atom(),
		Headers:         r.Headers(),
		ShardKey:        r.Atom(),
		RoutingKey:      r.Atom(),
		RoutingDelegate: r.Atom(),
		Body:            ioutil.NopCloser(bytes.NewReader(bodyData)),
	}, bodyData
}

func TestHash(t *testing.T) {
	rgen := newRandomGenerator(42)
	request, body := rgen.Request()

	recorder := NewRecorder(t)
	referenceHash := recorder.hashRequest(&request, body)

	require.Equal(t, "7195d5a712201d2a", referenceHash)

	// Caller
	r := request
	r.Caller = rgen.Atom()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// Service
	r = request
	r.Service = rgen.Atom()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// Encoding
	r = request
	r.Encoding = transport.Encoding(rgen.Atom())
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// Procedure
	r = request
	r.Procedure = rgen.Atom()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// Headers
	r = request
	r.Headers = rgen.Headers()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// ShardKey
	r = request
	r.ShardKey = rgen.Atom()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// RoutingKey
	r = request
	r.RoutingKey = rgen.Atom()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// RoutingDelegate
	r = request
	r.RoutingDelegate = rgen.Atom()
	assert.NotEqual(t, recorder.hashRequest(&r, body), referenceHash)

	// Body
	r = request
	b := []byte(rgen.Atom())
	assert.NotEqual(t, recorder.hashRequest(&r, b), referenceHash)
}

func TestOverwriteReplay(t *testing.T) {
	dir, err := ioutil.TempDir("", "yarpcgorecorder")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up

	func() {
		// First we double check that our cache is empty.
		recorder := NewRecorder(t, Option{
			Mode:        Replay,
			RecordsPath: dir,
		})

		clientDisp := yarpc.NewDispatcher(yarpc.Config{
			Name: "client",
			Outbounds: transport.Outbounds{
				"server": http.NewOutbound("http://localhost:65535"),
			},
			Filter: recorder,
		})
		require.NoError(t, clientDisp.Start())
		defer clientDisp.Stop()

		client := raw.New(clientDisp.Channel("server"))
		ctx, _ := context.WithTimeout(context.Background(), time.Second)

		_, _, err := client.Call(ctx, yarpc.NewReqMeta().Procedure("hello"), []byte("Hello"))
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	}()

	func() {
		// Now let's record our call.
		recorder := NewRecorder(t, Option{
			Mode:        Overwrite,
			RecordsPath: dir,
		})

		serverHTTP := http.NewInbound(":0")

		serverDisp := yarpc.NewDispatcher(yarpc.Config{
			Name:     "server",
			Inbounds: []transport.Inbound{serverHTTP},
		})

		serverDisp.Register(raw.Procedure("hello",
			func(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
				return append(body, []byte(", World")...), nil, nil
			}))

		require.NoError(t, serverDisp.Start())
		defer serverDisp.Stop()

		clientDisp := yarpc.NewDispatcher(yarpc.Config{
			Name: "client",
			Outbounds: transport.Outbounds{
				"server": http.NewOutbound(fmt.Sprintf("http://%s",
					serverHTTP.Addr().String())),
			},
			Filter: recorder,
		})
		require.NoError(t, clientDisp.Start())
		defer clientDisp.Stop()

		client := raw.New(clientDisp.Channel("server"))
		ctx, _ := context.WithTimeout(context.Background(), time.Second)

		rbody, _, err := client.Call(ctx, yarpc.NewReqMeta().Procedure("hello"), []byte("Hello"))
		require.NoError(t, err)
		assert.Equal(t, rbody, []byte("Hello, World"))
	}()

	func() {
		// Now we should be able to replay.
		recorder := NewRecorder(t, Option{
			Mode:        Replay,
			RecordsPath: dir,
		})

		clientDisp := yarpc.NewDispatcher(yarpc.Config{
			Name: "client",
			Outbounds: transport.Outbounds{
				"server": http.NewOutbound("http://localhost:65535"),
			},
			Filter: recorder,
		})
		require.NoError(t, clientDisp.Start())
		defer clientDisp.Stop()

		client := raw.New(clientDisp.Channel("server"))
		ctx, _ := context.WithTimeout(context.Background(), time.Second)

		rbody, _, err := client.Call(ctx, yarpc.NewReqMeta().Procedure("hello"), []byte("Hello"))
		require.NoError(t, err)
		assert.Equal(t, rbody, []byte("Hello, World"))
	}()
}
