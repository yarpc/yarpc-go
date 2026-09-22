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
	"bytes"
	"context"
	"io"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/wire"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"go.uber.org/yarpc/internal/clientconfig"
	yhttp "go.uber.org/yarpc/transport/http"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

const (
	// _goAwayPayloadSize is the size of the Thrift string in the request. It
	// must be much larger than one HTTP/2 frame, so the writer goroutine
	// needs many Read calls and is still working when the GOAWAY lands.
	_goAwayPayloadSize = 4 << 20

	// _goAwayAfterBytes is how much of the body the server takes before it
	// sends GOAWAY. Early, so most of the body is still unread.
	_goAwayAfterBytes = 128 << 10

	// _h2MaxFrameSize is what the test server advertises as
	// SETTINGS_MAX_FRAME_SIZE. The transport sizes its body read buffer as
	// min(peer max frame size, ContentLength+1, 512 KiB), so this makes each
	// body.Read copy 512 KiB instead of the default 16 KiB.
	//
	// This matters. With 16 KiB reads the writer goroutine spends almost all
	// of its time in the socket flush, and a concurrent Close is never
	// observable. With 512 KiB reads the writer spends about half its time
	// inside Read.
	_h2MaxFrameSize = 1 << 20

	// _goAwayAttempts is how many times the test always runs the scenario.
	//
	// Every attempt gives the pool churn another chance to take a buffer that
	// went back too early, so the race detector gets more chances to see it.
	// The test therefore runs this many attempts even after it has what it
	// needs.
	_goAwayAttempts = 4

	// _goAwayMaxAttempts bounds the extra attempts the test makes when it has
	// not yet seen the overlap.
	//
	// The overlap is itself a race: when the GOAWAY is processed, the writer
	// goroutine is either inside Read or inside the socket flush, and only
	// the first case is observable. It happens about half the time, so a
	// fixed 4 attempts fails about 1 run in 20. Going on to 12 makes that
	// about 1 run in 4000, and costs nothing on a normal run.
	_goAwayMaxAttempts = 12

	// _goAwayReplyBody is the Thrift string the server answers with.
	_goAwayReplyBody = "response"
)

// TestNoWireClient_SingleRequestGoAwayMidBody drives ONE outbound Thrift
// NoWire call, with a large request body, against a real HTTP/2 server that
// sends GOAWAY while the client is still writing that body.
//
// One request is enough. The GOAWAY makes the transport close the pooled
// body on its own goroutine while another goroutine still reads it.
//
// The test asserts that Close overlaps the read loop, that a Read after Close
// is refused, and that the transport asks for a replay and the call still
// succeeds.
//
// The overlap is itself a race: when the GOAWAY lands, the writer is either
// inside Read or inside the socket flush, and only the first is observable.
// The test therefore repeats the scenario. Every attempt is a complete, valid
// request, so repeating weakens no assertion.
func TestNoWireClient_SingleRequestGoAwayMidBody(t *testing.T) {
	payload := strings.Repeat("x", _goAwayPayloadSize)

	// Run every attempt. Keep the first one that showed the overlap, and fall
	// back to the last one so that the diagnostics below always have data.
	var (
		got         goAwayAttempt
		overlapping goAwayAttempt
		overlapAt   int
		attemptsRun int
	)
	for attempt := 1; attempt <= _goAwayMaxAttempts; attempt++ {
		attemptsRun = attempt
		got = runGoAwayMidBodyAttempt(t, payload)
		if got.sawOverlap() && overlapAt == 0 {
			overlapping, overlapAt = got, attempt
		}
		// Always run the first _goAwayAttempts. Go on past them only to look
		// for an overlap that has not shown up yet.
		if attempt >= _goAwayAttempts && overlapAt != 0 {
			break
		}
	}
	if overlapAt != 0 {
		got = overlapping
	}

	ob := got.observer
	require.NotNil(t, ob, "the middleware must have seen a request body")

	t.Logf("attempts run:                 %d", attemptsRun)
	t.Logf("first attempt with overlap:   %d", overlapAt)
	t.Logf("encoded request body:         %d bytes", ob.bodySize.Load())
	t.Logf("bytes the first attempt sent: %d", got.server.firstBodyBytes.Load())
	t.Logf("connections the client made:  %d", got.server.conns.Load())
	t.Logf("Read calls:                   %d", ob.reads.Load())
	t.Logf("bytes the writer read:        %d", ob.bytesRead.Load())
	t.Logf("Close calls:                  %d", ob.closes.Load())
	t.Logf("Reads that began after Close: %d", ob.readsAfterClose.Load())
	t.Logf("Close arrived during a Read:  %v", ob.closeDuringRead.Load())
	t.Logf("GetBody calls:                %d", ob.getBodyCalls.Load())
	if e := ob.errAfterClose.Load(); e != nil {
		t.Logf("error the late read got:      %v", e)
	}

	// --- the scenario really happened ---

	require.True(t, got.server.sentGoAway.Load(),
		"the server must have sent GOAWAY mid-body; otherwise this test proves nothing")
	require.Positive(t, ob.closes.Load(),
		"the transport must close the request body")
	assert.Less(t, ob.bytesRead.Load(), ob.bodySize.Load(),
		"the writer must have been stopped mid-body, not after draining the whole body")

	// --- Close overlaps the writer's read loop ---

	// Close runs on a goroutine of its own and does not stop the writer. The
	// overlap shows itself in one of two ways:
	//
	//   - Close arrives while a Read is running, or
	//   - the writer starts one more Read after Close returned.
	//
	// This is the moment the body and the pool have to agree on who owns the
	// bytes.
	require.True(t, got.sawOverlap(),
		"Close must overlap the writer's read loop in at least 1 of %d attempts",
		attemptsRun)

	// --- a Read after Close is refused ---

	if ob.readsAfterClose.Load() > 0 {
		e, _ := ob.errAfterClose.Load().(error)
		require.ErrorIs(t, e, ErrRequestBodyReleased,
			"a Read after Close must be refused, not served bytes")
	}

	// --- the replay: the transport resends the request and the call succeeds ---

	// GetBody encodes the request again, into a new buffer from the pool. It
	// does not need the first buffer, which Close already returned.
	assert.Positive(t, ob.getBodyCalls.Load(),
		"the transport must ask GetBody for a replay after GOAWAY")
	assert.Greater(t, got.server.conns.Load(), int32(1),
		"the replay must go out on a new connection")
	require.NoError(t, got.err,
		"the call must survive the GOAWAY")
	assert.Equal(t, _goAwayReplyBody, got.responseBody,
		"the replayed request must get the whole reply")
}

// goAwayAttempt is the result of one run of the scenario.
type goAwayAttempt struct {
	observer     *bodyObserver
	server       *midBodyGoAwayServer
	err          error
	responseBody string
}

func (a goAwayAttempt) sawOverlap() bool {
	if a.observer == nil {
		return false
	}
	return a.observer.closeDuringRead.Load() || a.observer.readsAfterClose.Load() > 0
}

// runGoAwayMidBodyAttempt performs one complete call against a fresh server
// and a fresh outbound, and reports what the transport did to the body.
func runGoAwayMidBodyAttempt(t *testing.T, payload string) goAwayAttempt {
	t.Helper()

	server := newMidBodyGoAwayServer(t, encodeThriftString(t, _goAwayReplyBody))
	defer server.close()

	httpTransport := yhttp.NewTransport()
	require.NoError(t, httpTransport.Start(), "failed to start the http transport")
	defer func() { assert.NoError(t, httpTransport.Stop()) }()

	out := httpTransport.NewSingleOutbound("http://"+server.addr(), yhttp.UseHTTP2())
	require.NoError(t, out.Start(), "failed to start the http2 outbound")
	defer func() { assert.NoError(t, out.Stop()) }()

	observer := &bodyObserverMiddleware{}
	nwc := NewNoWire(Config{
		Service: "MyService",
		ClientConfig: clientconfig.MultiOutbound("caller", "service",
			transport.Outbounds{
				Unary: middleware.ApplyUnaryOutbound(out, observer),
			}),
	})

	// Any other traffic in the process shares the global bufferpool. It takes
	// any buffer that goes back too early and writes its own data into it, so
	// a read of those bytes becomes a reported race.
	//
	// This is what makes the race visible with a single request. The pool's
	// own use-after-free detection cannot do it: bufferpool.init runs before
	// testing.Init registers -test.v, so the global pool always has
	// testDetectUseAfterFree false, even under go test.
	stopChurn := startPoolChurn(len(payload))
	defer stopChurn()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var br fakeBodyReader
	err := nwc.Call(ctx, largeEnveloper{payload: payload}, &br)

	// Let any goroutine the transport left running finish, so its reads are
	// counted here and so the race detector sees them.
	time.Sleep(100 * time.Millisecond)

	return goAwayAttempt{
		observer:     observer.first(),
		server:       server,
		err:          err,
		responseBody: br.body,
	}
}

// startPoolChurn keeps taking buffers from the global pool, writing to them,
// and giving them back, until the returned function is called. It stands in
// for all the other outbound traffic in a real process.
//
// If the request body's buffer goes back to the pool while the HTTP/2 writer
// still reads it, this loop takes that buffer and writes over those bytes.
// go test -race then reports the read and the write as a data race.
func startPoolChurn(size int) func() {
	var (
		stop    atomic.Bool
		wg      sync.WaitGroup
		workers = runtime.GOMAXPROCS(0)
	)
	if workers < 2 {
		workers = 2
	}
	if workers > 4 {
		workers = 4
	}

	// One worker per P. sync.Pool keeps per-P free lists, so a single worker
	// often never sees the buffer that another P released.
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pattern := bytes.Repeat([]byte{0xAB}, size)
			for !stop.Load() {
				buf := bufferpool.Get()
				_, _ = buf.Write(pattern)
				bufferpool.Put(buf)
			}
		}()
	}

	return func() {
		stop.Store(true)
		wg.Wait()
	}
}

// largeEnveloper encodes one large Thrift string, so the pooled request
// buffer spans many HTTP/2 frames.
type largeEnveloper struct{ payload string }

func (largeEnveloper) MethodName() string { return "someMethod" }

func (largeEnveloper) EnvelopeType() wire.EnvelopeType { return wire.Call }

func (largeEnveloper) ToWire() (wire.Value, error) {
	return wire.NewValueStruct(wire.Struct{}), nil
}

func (e largeEnveloper) Encode(sw stream.Writer) error { return sw.WriteString(e.payload) }

// --- observation of the request body ---

// bodyObserver counts what the transport does to one request body. It changes
// no behaviour; it only records.
type bodyObserver struct {
	inner io.ReadCloser

	reads           atomic.Int32
	bytesRead       atomic.Int64
	bodySize        atomic.Int64
	readsInFlight   atomic.Int32
	closes          atomic.Int32
	closed          atomic.Bool
	closeDuringRead atomic.Bool
	readsAfterClose atomic.Int32
	errAfterClose   atomic.Value
	getBodyCalls    atomic.Int32
}

func (o *bodyObserver) Read(p []byte) (int, error) {
	afterClose := o.closed.Load()
	if afterClose {
		o.readsAfterClose.Add(1)
	}

	o.readsInFlight.Add(1)
	n, err := o.inner.Read(p)
	o.readsInFlight.Add(-1)

	o.reads.Add(1)
	o.bytesRead.Add(int64(n))
	if afterClose && err != nil {
		o.errAfterClose.CompareAndSwap(nil, err)
	}
	return n, err
}

func (o *bodyObserver) Close() error {
	if o.readsInFlight.Load() > 0 {
		o.closeDuringRead.Store(true)
	}
	o.closes.Add(1)

	err := o.inner.Close()

	// Mark the body closed only after the real Close has returned. A Read
	// that starts before that point is concurrent with Close, not after it,
	// and the body may still serve it. closeDuringRead records that case.
	// Without this order, readsAfterClose can count a Read that legitimately
	// got bytes and a nil error.
	o.closed.Store(true)
	return err
}

// bodyObserverRewinder adds GetBody, so that a body which can rewind itself
// keeps that property through the observer. Without this the observer would
// change which branch transport/http takes, and the test would measure the
// wrong code path.
type bodyObserverRewinder struct {
	*bodyObserver
	rewinder interface {
		GetBody() (io.ReadCloser, error)
	}
}

func (o *bodyObserverRewinder) GetBody() (io.ReadCloser, error) {
	o.getBodyCalls.Add(1)
	return o.rewinder.GetBody()
}

// bodyObserverMiddleware installs a bodyObserver on the outbound request
// body, and keeps the first one for the test to read.
type bodyObserverMiddleware struct{ observed atomic.Value }

func (m *bodyObserverMiddleware) first() *bodyObserver {
	v := m.observed.Load()
	if v == nil {
		return nil
	}
	return v.(*bodyObserver)
}

func (m *bodyObserverMiddleware) Call(ctx context.Context, req *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	rc, ok := req.Body.(io.ReadCloser)
	if !ok {
		rc = io.NopCloser(req.Body)
	}

	ob := &bodyObserver{inner: rc}
	ob.bodySize.Store(int64(req.BodySize))
	m.observed.CompareAndSwap(nil, ob)

	if rw, ok := req.Body.(interface {
		GetBody() (io.ReadCloser, error)
	}); ok {
		req.Body = &bodyObserverRewinder{bodyObserver: ob, rewinder: rw}
	} else {
		req.Body = ob
	}
	return out.Call(ctx, req)
}

var _ middleware.UnaryOutbound = (*bodyObserverMiddleware)(nil)

// --- an HTTP/2 server that sends GOAWAY in the middle of the request body ---

// midBodyGoAwayServer speaks cleartext HTTP/2. The first connection to reach
// _goAwayAfterBytes of request body sends GOAWAY, while the client is still
// writing. Every later connection accepts the whole request and answers with
// the given Thrift response.
//
// The existing h2cGoAwayServer in transport/http waits for END_STREAM before
// it sends GOAWAY. That is too late for this test: the writer goroutine has
// already finished, so no read and no close can overlap.
type midBodyGoAwayServer struct {
	listener net.Listener
	response string

	conns           atomic.Int32
	sentGoAway      atomic.Bool
	firstBodyBytes  atomic.Int64
	replayBodyBytes atomic.Int64
}

func newMidBodyGoAwayServer(tb testing.TB, response string) *midBodyGoAwayServer {
	tb.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(tb, err, "failed to listen for the h2c server")

	s := &midBodyGoAwayServer{listener: ln, response: response}
	go s.serve()
	return s
}

func (s *midBodyGoAwayServer) addr() string { return s.listener.Addr().String() }

func (s *midBodyGoAwayServer) close() { _ = s.listener.Close() }

func (s *midBodyGoAwayServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.conns.Add(1)
		go s.handleConn(conn)
	}
}

func (s *midBodyGoAwayServer) handleConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	preface := make([]byte, len(http2.ClientPreface))
	if _, err := io.ReadFull(conn, preface); err != nil {
		return
	}

	framer := http2.NewFramer(conn, conn)
	framer.SetMaxReadFrameSize(_h2MaxFrameSize)

	// Give the client a very large flow-control window, and a large frame
	// size. The writer goroutine must not park in awaitFlowControl: a writer
	// that parks there exits through awaitFlowControl after the GOAWAY and
	// never calls Read again, which is the overlap this test must observe.
	if err := framer.WriteSettings(
		http2.Setting{ID: http2.SettingInitialWindowSize, Val: 1<<30 - 1},
		http2.Setting{ID: http2.SettingMaxFrameSize, Val: _h2MaxFrameSize},
	); err != nil {
		return
	}
	if err := framer.WriteWindowUpdate(0, 1<<30); err != nil {
		return
	}

	var (
		received   int
		goAwayHere bool
		deadline   = time.Now().Add(30 * time.Second)
	)
	for time.Now().Before(deadline) {
		frame, err := framer.ReadFrame()
		if err != nil {
			return
		}

		switch f := frame.(type) {
		case *http2.SettingsFrame:
			if !f.IsAck() {
				if err := framer.WriteSettingsAck(); err != nil {
					return
				}
			}

		case *http2.HeadersFrame:
			if f.StreamEnded() {
				_ = s.writeResponse(framer, f.StreamID)
				return
			}

		case *http2.DataFrame:
			received += len(f.Data())

			if goAwayHere {
				// Keep draining after the GOAWAY. The socket has to stay
				// readable for the client's writer to make progress into the
				// read that overlaps the Close.
				if f.StreamEnded() {
					return
				}
				continue
			}

			// The first connection to take _goAwayAfterBytes sends the
			// GOAWAY. Do not key this on a connection number: the client may
			// open a connection that it never uses.
			if received >= _goAwayAfterBytes && s.sentGoAway.CompareAndSwap(false, true) {
				s.firstBodyBytes.Store(int64(received))
				goAwayHere = true
				// LastStreamID 0 tells the client that no stream was
				// processed, so the transport replays it on a new connection.
				if err := framer.WriteGoAway(0, http2.ErrCodeNo, nil); err != nil {
					return
				}
				continue
			}

			if f.StreamEnded() {
				s.replayBodyBytes.Store(int64(received))
				_ = s.writeResponse(framer, f.StreamID)
				return
			}
		}
	}
}

// writeResponse answers with 200 and the Thrift payload.
func (s *midBodyGoAwayServer) writeResponse(framer *http2.Framer, streamID uint32) error {
	var buf bytes.Buffer
	if err := hpack.NewEncoder(&buf).WriteField(hpack.HeaderField{
		Name:  ":status",
		Value: "200",
	}); err != nil {
		return err
	}
	if err := framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamID,
		BlockFragment: buf.Bytes(),
		EndHeaders:    true,
	}); err != nil {
		return err
	}
	return framer.WriteData(streamID, true, []byte(s.response))
}
