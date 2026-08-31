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

package http

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/testtime"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

// TestHTTP2GoAwayReplay exercises HTTP/2 transport implementation w.r.t. GOAWAY frames.
func TestHTTP2GoAwayReplay(t *testing.T) {
	tests := []struct {
		name    string
		rejects int32 // Number of requests the server rejects with GOAWAY
		mw      middleware.UnaryOutbound
		reqBody io.Reader
		wantErr bool
	}{
		{
			name:    "no GOAWAY: no-op middleware, valid Request.Body; no replay needed",
			rejects: 0,
			mw:      middleware.NopUnaryOutbound,
			reqBody: getH2ValidBody(),
			wantErr: false,
		},
		{
			name:    "no GOAWAY: no-op middleware, invalid Request.Body; no replay needed",
			rejects: 0,
			mw:      middleware.NopUnaryOutbound,
			reqBody: getH2InvalidBody(),
			wantErr: false,
		},
		{
			name:    "one GOAWAY: no-op middleware, valid Request.Body; replay succeeds",
			rejects: 1, // 1 means reject the first request with GOAWAY, then accept the next ones
			mw:      middleware.NopUnaryOutbound,
			reqBody: getH2ValidBody(),
			wantErr: false,
		},
		{
			name:    "one GOAWAY: no-op middleware, invalid Request.Body; replay succeeds",
			rejects: 1, // 1 means reject the first request with GOAWAY, then accept the next ones
			mw:      middleware.NopUnaryOutbound,
			reqBody: getH2InvalidBody(),
			wantErr: false,
		},
		{
			name:    "one GOAWAY: custom middleware, valid Request.Body becomes invalid; replay succeeds",
			rejects: 1, // 1 means reject the first request with GOAWAY, then accept the next ones
			mw:      someCustomMiddleware{},
			reqBody: getH2ValidBody(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newH2CGoAwayServer(t, tt.rejects)
			outboundWithMiddleware := makeYarpcOutbound(t, server, tt.mw)

			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			t.Cleanup(cancel)

			_, err := outboundWithMiddleware.Call(ctx, &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  raw.Encoding,
				Procedure: "some-procedure",
				Body:      tt.reqBody,
			})

			if tt.wantErr {
				require.Error(t, err, "expected replay to fail when GetBody is not set")
				assert.Contains(t, err.Error(), "GetBody", "error should mention GetBody")
				return
			}
			require.NoError(t, err, "expected replay to succeed when GetBody is set")
		})
	}
}

// BenchmarkHTTP2GoAway exercises HTTP/2 transport implementation w.r.t. GOAWAY frames.
// In particular, it focuses on net/http.Request creation, reads, and allocations in cases where an
// HTTP/2 server sends GOAWAY frames.
func BenchmarkHTTP2GoAway(b *testing.B) {
	var (
		blackholeResp *transport.Response
		blackholeErr  error
	)

	benchs := []struct {
		name    string
		rejects int32 // Number of requests the server rejects with GOAWAY
		mw      middleware.UnaryOutbound
		reqBody io.Reader
	}{
		{
			name:    "happy path: no-op middleware, no GOAWAY, valid Request.Body",
			rejects: 0,
			mw:      middleware.NopUnaryOutbound,
			reqBody: getH2ValidBody(),
		},
		{
			name:    "happy path: no-op middleware, no GOAWAY, invalid Request.Body",
			rejects: 0,
			mw:      middleware.NopUnaryOutbound,
			reqBody: getH2InvalidBody(),
		},
		{
			name:    "sad path: no-op middleware, one GOAWAY, valid Request.Body",
			rejects: 1, // 1 means reject the first request with GOAWAY, then accept the next ones
			mw:      middleware.NopUnaryOutbound,
			reqBody: getH2ValidBody(),
		},
		{
			name:    "sad path: no-op middleware, one GOAWAY, invalid Request.Body",
			rejects: 1, // 1 means reject the first request with GOAWAY, then accept the next ones
			mw:      middleware.NopUnaryOutbound,
			reqBody: getH2InvalidBody(),
		},
		{
			name:    "sad path: custom middleware, one GOAWAY, valid Request.Body becomes invalid",
			rejects: 1, // 1 means reject the first request with GOAWAY, then accept the next ones
			mw:      someCustomMiddleware{},
			reqBody: getH2ValidBody(),
		},
		{
			name:    "saddest path: no-op middleware, always GOAWAY, valid Request.Body",
			rejects: -1, // -1 means always reject with GOAWAY
			mw:      middleware.NopUnaryOutbound,
			reqBody: getH2ValidBody(),
		},
		{
			name:    "saddest path: no-op middleware, always GOAWAY, invalid Request.Body",
			rejects: -1, // -1 means always reject with GOAWAY
			mw:      middleware.NopUnaryOutbound,
			reqBody: getH2InvalidBody(),
		},
		{
			name:    "saddest path: custom middleware, always GOAWAY, valid Request.Body becomes invalid",
			rejects: -1, // -1 means always reject with GOAWAY
			mw:      someCustomMiddleware{},
			reqBody: getH2ValidBody(),
		},
	}
	for _, bb := range benchs {
		b.Run(bb.name, func(b *testing.B) {
			server := newH2CGoAwayServer(b, bb.rejects)
			outboundWithMiddleware := makeYarpcOutbound(b, server, bb.mw)

			ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
			b.Cleanup(cancel)

			treq := &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  raw.Encoding,
				Procedure: "some-procedure",
				Body:      bb.reqBody,
			}

			b.ResetTimer()
			for range b.N {
				blackholeResp, blackholeErr = outboundWithMiddleware.Call(ctx, treq)
			}
			b.StopTimer()
		})
	}

	_ = blackholeResp
	_ = blackholeErr
}

func makeYarpcOutbound(tb testing.TB, s *h2cGoAwayServer, mw middleware.UnaryOutbound) transport.UnaryOutbound {
	tb.Helper()

	httpTransport := NewTransport()
	require.NoError(tb, httpTransport.Start(), "failed to start http transport")
	tb.Cleanup(func() { assert.NoError(tb, httpTransport.Stop()) })

	out := httpTransport.NewSingleOutbound("http://"+s.addr(), UseHTTP2())
	require.NoError(tb, out.Start(), "failed to start http2 outbound")
	tb.Cleanup(func() {
		assert.NoError(tb, out.Stop())
	})

	return middleware.ApplyUnaryOutbound(out, mw)
}

func getH2ValidBody() io.Reader {
	// transport.Request.Body must be one of *bytes.Buffer, *bytes.Reader, or *strings.Reader.
	return strings.NewReader("some-payload")
}

func getH2InvalidBody() io.Reader {
	// Any other transport.Request.Body concrete type that does not allow automatic GetBody creation.
	return &someCustomBody{reader: strings.NewReader("some-payload")}
}

// --- HTTP/2 server: send a configurable number of GOAWAY frames ---

// h2cGoAwayServer is a minimal cleartext HTTP/2 server that sends a GOAWAY on
// to a configurable number of requests and answers with 200 to the rest.
// This reproduces the net/http2 internal replay path that requires
// http.Request.GetBody to be set.
type h2cGoAwayServer struct {
	listener     net.Listener
	requests     atomic.Int32
	rejectBefore int32 // Number of requests to reject with GOAWAY before accepting the rest; -1 rejects all requests.
}

func newH2CGoAwayServer(tb testing.TB, rejectBefore int32) *h2cGoAwayServer {
	tb.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(tb, err, "failed to listen for the h2c server")

	if rejectBefore < -1 {
		rejectBefore = -1
	}
	s := &h2cGoAwayServer{
		listener:     ln,
		rejectBefore: rejectBefore,
	}
	go s.serve()
	tb.Cleanup(func() { _ = ln.Close() })
	return s
}

func (s *h2cGoAwayServer) addr() string { return s.listener.Addr().String() }

func (s *h2cGoAwayServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *h2cGoAwayServer) handleConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	preface := make([]byte, len(http2.ClientPreface))
	if _, err := io.ReadFull(conn, preface); err != nil {
		return
	}
	framer := http2.NewFramer(conn, conn)
	if err := framer.WriteSettings(); err != nil {
		return
	}

	streamID, ok := awaitFullRequest(framer)
	if !ok {
		return
	}

	// Send GOAWAY to any LastStreamID (-1) or LastStreamID <= s.rejectBefore so
	// net/http2 considers stream "not processed" and replays it on a new connection.
	if s.rejectBefore == -1 || s.requests.Add(1) <= s.rejectBefore {
		_ = framer.WriteGoAway(0, http2.ErrCodeNo, nil)
		return
	}
	_ = writeH2Status(framer, streamID, "200")
}

// awaitFullRequest reads frames until the client finishes sending a request
// (END_STREAM), acking SETTINGS along the way.
func awaitFullRequest(framer *http2.Framer) (streamID uint32, ok bool) {
	for {
		frame, err := framer.ReadFrame()
		if err != nil {
			return 0, false
		}
		switch f := frame.(type) {
		case *http2.SettingsFrame:
			if !f.IsAck() {
				_ = framer.WriteSettingsAck()
			}
		case *http2.HeadersFrame:
			if f.StreamEnded() {
				return f.StreamID, true
			}
		case *http2.DataFrame:
			if f.StreamEnded() {
				return f.StreamID, true
			}
		}
	}
}

// writeH2Status writes a HEADERS frame with :status and END_STREAM.
func writeH2Status(framer *http2.Framer, streamID uint32, status string) error {
	var buf bytes.Buffer
	if err := hpack.NewEncoder(&buf).WriteField(hpack.HeaderField{Name: ":status", Value: status}); err != nil {
		return err
	}
	return framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamID,
		BlockFragment: buf.Bytes(),
		EndStream:     true,
		EndHeaders:    true,
	})
}

// --- YARPC outbound middleware manipulating transport.Request.Body ---

// someCustomBody represents a custom transport.Request.Body field type that Go net/http does not match when automatically providing a http.Request.GetBody function.
type someCustomBody struct{ reader io.Reader }

func (b someCustomBody) Read(p []byte) (int, error) { return b.reader.Read(p) }

// someCustomMiddleware is an outbound middleware that manipulates transport.Request.Body to be of type someCustomBody.
type someCustomMiddleware struct{}

// Call implements the middleware.UnaryOutbound interface.
// This mimics a middleware that manipulates transport.Request.Body into a type that does not let GetBody be automatically populated by net/http.
func (someCustomMiddleware) Call(ctx context.Context, req *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	req.Body = &someCustomBody{reader: req.Body}
	return out.Call(ctx, req)
}

var (
	_ io.Reader                = someCustomBody{}
	_ middleware.UnaryOutbound = someCustomMiddleware{}
)
