// Copyright (c) 2020 Uber Technologies, Inc.
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

package recorder

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/yarpctest"
	"go.uber.org/yarpc/transport/http"
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
func (r *randomGenerator) Request() transport.Request {
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
		Body:            bytes.NewReader(bodyData),
	}
}

func TestHash(t *testing.T) {
	rgen := newRandomGenerator(42)
	request := rgen.Request()

	recorder := NewRecorder(t)
	requestRecord := recorder.requestToRequestRecord(&request)
	referenceHash := recorder.hashRequestRecord(&requestRecord)

	require.Equal(t, "7195d5a712201d2a", referenceHash)

	// Caller
	r := request
	r.Caller = rgen.Atom()
	requestRecord = recorder.requestToRequestRecord(&r)
	assert.NotEqual(t, recorder.hashRequestRecord(&requestRecord), referenceHash)

	// Service
	r = request
	r.Service = rgen.Atom()
	requestRecord = recorder.requestToRequestRecord(&r)
	assert.NotEqual(t, recorder.hashRequestRecord(&requestRecord), referenceHash)

	// Encoding
	r = request
	r.Encoding = transport.Encoding(rgen.Atom())
	requestRecord = recorder.requestToRequestRecord(&r)
	assert.NotEqual(t, recorder.hashRequestRecord(&requestRecord), referenceHash)

	// Procedure
	r = request
	r.Procedure = rgen.Atom()
	requestRecord = recorder.requestToRequestRecord(&r)
	assert.NotEqual(t, recorder.hashRequestRecord(&requestRecord), referenceHash)

	// Headers
	r = request
	r.Headers = rgen.Headers()
	requestRecord = recorder.requestToRequestRecord(&r)
	assert.NotEqual(t, recorder.hashRequestRecord(&requestRecord), referenceHash)

	// ShardKey
	r = request
	r.ShardKey = rgen.Atom()
	requestRecord = recorder.requestToRequestRecord(&r)
	assert.NotEqual(t, recorder.hashRequestRecord(&requestRecord), referenceHash)

	// RoutingKey
	r = request
	r.RoutingKey = rgen.Atom()
	requestRecord = recorder.requestToRequestRecord(&r)
	assert.NotEqual(t, recorder.hashRequestRecord(&requestRecord), referenceHash)

	// RoutingDelegate
	r = request
	r.RoutingDelegate = rgen.Atom()
	requestRecord = recorder.requestToRequestRecord(&r)
	assert.NotEqual(t, recorder.hashRequestRecord(&requestRecord), referenceHash)

	// Body
	r = request
	request.Body = bytes.NewReader([]byte(rgen.Atom()))
	requestRecord = recorder.requestToRequestRecord(&r)
	assert.NotEqual(t, recorder.hashRequestRecord(&requestRecord), referenceHash)
}

var testingTMockFatal = struct{}{}

type testingTMock struct {
	*testing.T

	fatalCount int
}

func (t *testingTMock) Fatal(args ...interface{}) {
	t.Logf("counting fatal: %s", args)
	t.fatalCount++
	panic(testingTMockFatal)
}

func withDisconnectedClient(t *testing.T, recorder *Recorder, f func(raw.Client)) {
	httpTransport := http.NewTransport()

	clientDisp := yarpc.NewDispatcher(yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"server": {
				Unary: httpTransport.NewSingleOutbound("http://127.0.0.1:65535"),
			},
		},
		OutboundMiddleware: yarpc.OutboundMiddleware{
			Unary: recorder,
		},
	})
	require.NoError(t, clientDisp.Start())
	defer clientDisp.Stop()

	client := raw.New(clientDisp.ClientConfig("server"))
	f(client)
}

func withConnectedClient(t *testing.T, recorder *Recorder, f func(raw.Client)) {
	httpTransport := http.NewTransport()
	serverHTTP := httpTransport.NewInbound("127.0.0.1:0")
	serverDisp := yarpc.NewDispatcher(yarpc.Config{
		Name:     "server",
		Inbounds: yarpc.Inbounds{serverHTTP},
	})

	serverDisp.Register(raw.Procedure("hello",
		func(ctx context.Context, body []byte) ([]byte, error) {
			return append(body, []byte(", World")...), nil
		}))

	require.NoError(t, serverDisp.Start())
	defer serverDisp.Stop()

	clientDisp := yarpc.NewDispatcher(yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"server": {
				Unary: httpTransport.NewSingleOutbound(fmt.Sprintf("http://%s", yarpctest.ZeroAddrToHostPort(serverHTTP.Addr()))),
			},
		},
		OutboundMiddleware: yarpc.OutboundMiddleware{
			Unary: recorder,
		},
	})
	require.NoError(t, clientDisp.Start())
	defer clientDisp.Stop()

	client := raw.New(clientDisp.ClientConfig("server"))
	f(client)
}

func TestEndToEnd(t *testing.T) {
	tMock := testingTMock{t, 0}

	dir, err := ioutil.TempDir("", "yarpcgorecorder")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up

	// First we double check that our cache is empty.
	recorder := NewRecorder(&tMock, RecordMode(Replay), RecordsPath(dir))

	withDisconnectedClient(t, recorder, func(client raw.Client) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		require.Panics(t, func() {
			client.Call(ctx, "hello", []byte("Hello"))
		})
		assert.Equal(t, tMock.fatalCount, 1)
	})

	// Now let's record our call.
	recorder = NewRecorder(&tMock, RecordMode(Overwrite), RecordsPath(dir))

	withConnectedClient(t, recorder, func(client raw.Client) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		rbody, err := client.Call(ctx, "hello", []byte("Hello"))
		require.NoError(t, err)
		assert.Equal(t, rbody, []byte("Hello, World"))
	})

	// Now replay the call.
	recorder = NewRecorder(&tMock, RecordMode(Replay), RecordsPath(dir))

	withDisconnectedClient(t, recorder, func(client raw.Client) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		rbody, err := client.Call(ctx, "hello", []byte("Hello"))
		require.NoError(t, err)
		assert.Equal(t, rbody, []byte("Hello, World"))
	})
}

func TestEmptyReplay(t *testing.T) {
	tMock := testingTMock{t, 0}

	dir, err := ioutil.TempDir("", "yarpcgorecorder")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up

	recorder := NewRecorder(&tMock, RecordMode(Replay), RecordsPath(dir))

	withDisconnectedClient(t, recorder, func(client raw.Client) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		require.Panics(t, func() {
			client.Call(ctx, "hello", []byte("Hello"))
		})
		assert.Equal(t, tMock.fatalCount, 1)
	})
}

const refRecordFilename = `server.hello.254fa3bab61fc27f.yaml`
const refRecordContent = recordComment +
	`version: 1
request:
  caller: client
  service: server
  procedure: hello
  encoding: raw
  headers: {}
  shardkey: ""
  routingkey: ""
  routingdelegate: ""
  body: SGVsbG8=
response:
  headers: {}
  body: SGVsbG8sIFdvcmxk
`

func TestRecording(t *testing.T) {
	tMock := testingTMock{t, 0}

	dir, err := ioutil.TempDir("", "yarpcgorecorder")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up

	recorder := NewRecorder(&tMock, RecordMode(Append), RecordsPath(dir))

	withConnectedClient(t, recorder, func(client raw.Client) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		rbody, err := client.Call(ctx, "hello", []byte("Hello"))
		require.NoError(t, err)
		assert.Equal(t, []byte("Hello, World"), rbody)
	})

	recordPath := path.Join(dir, refRecordFilename)
	_, err = os.Stat(recordPath)
	require.NoError(t, err)

	recordContent, err := ioutil.ReadFile(recordPath)
	require.NoError(t, err)
	assert.Equal(t, refRecordContent, string(recordContent))
}

func TestReplaying(t *testing.T) {
	tMock := testingTMock{t, 0}

	dir, err := ioutil.TempDir("", "yarpcgorecorder")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up

	recorder := NewRecorder(&tMock, RecordMode(Replay), RecordsPath(dir))

	recordPath := path.Join(dir, refRecordFilename)
	err = ioutil.WriteFile(recordPath, []byte(refRecordContent), 0444)
	require.NoError(t, err)

	withDisconnectedClient(t, recorder, func(client raw.Client) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		rbody, err := client.Call(ctx, "hello", []byte("Hello"))
		require.NoError(t, err)
		assert.Equal(t, rbody, []byte("Hello, World"))
	})
}
