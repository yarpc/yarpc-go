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
	"go.uber.org/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/middleware"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/testtime"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

// --- HTTP/2 server: immediately send GOAWAY frame ---

// h2cGoAwayServer is a minimal cleartext HTTP/2 server that sends a GOAWAY on
// the first request and answers subsequent requests with 200. This reproduces
// the net/http2 internal replay path that requires http.Request.GetBody to be
// set.
type h2cGoAwayServer struct {
	listener net.Listener
	requests atomic.Int32
}

func newH2CGoAwayServer(t *testing.T) *h2cGoAwayServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to listen for the h2c server")

	s := &h2cGoAwayServer{listener: ln}
	go s.serve()
	t.Cleanup(func() { _ = ln.Close() })
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

	// First request ever: send GOAWAY with LastStreamID=0 so net/http2 considers
	// stream 1 "not processed" and replays it on a new connection.
	if s.requests.Add(1) == 1 {
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

// useCustomBodyMiddleware is an outbound middleware that manipulates transport.Request.Body to be of type someCustomBody.
type useCustomBodyMiddleware struct{}

func (useCustomBodyMiddleware) Call(ctx context.Context, req *transport.Request, out transport.UnaryOutbound) (*transport.Response, error) {
	req.Body = someCustomBody{reader: req.Body}
	return out.Call(ctx, req)
}

// --- ----- ----- ---

// TestHTTP2GoAwayReplay is a test suite covering HTTP/2 transport implementation w.r.t. GOAWAY frames.
func TestHTTP2GoAwayReplay(t *testing.T) {
	t.Run("no middleware - body type preserved - replay succeeds", func(t *testing.T) {
		server := newH2CGoAwayServer(t)

		httpTransport := NewTransport()
		require.NoError(t, httpTransport.Start(), "failed to start http transport")
		t.Cleanup(func() { assert.NoError(t, httpTransport.Stop()) })

		out := httpTransport.NewSingleOutbound("http://"+server.addr(), UseHTTP2())
		require.NoError(t, out.Start(), "failed to start http2 outbound")
		t.Cleanup(func() {
			assert.NoError(t, out.Stop())
			out.client.CloseIdleConnections()
		})

		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		t.Cleanup(cancel)

		// strings.NewReader is recognized by net/http, so GetBody is populated
		// and net/http2 can replay the request after the GOAWAY.
		res, err := out.Call(ctx, &transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  raw.Encoding,
			Procedure: "hello",
			Body:      strings.NewReader("world"),
		})
		require.NoError(t, err, "expected net/http2 to replay the request after GOAWAY when GetBody is set")
		_ = res.Body.Close()
	})

	t.Run("middleware wraps body - non-rewindable type - replay fails", func(t *testing.T) {
		server := newH2CGoAwayServer(t)

		httpTransport := NewTransport()
		require.NoError(t, httpTransport.Start(), "failed to start http transport")
		t.Cleanup(func() { assert.NoError(t, httpTransport.Stop()) })

		out := httpTransport.NewSingleOutbound("http://"+server.addr(), UseHTTP2())
		require.NoError(t, out.Start(), "failed to start http2 outbound")
		t.Cleanup(func() {
			assert.NoError(t, out.Stop())
			out.client.CloseIdleConnections()
		})

		outboundWithMiddleware := middleware.ApplyUnaryOutbound(out, useCustomBodyMiddleware{})

		ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
		t.Cleanup(cancel)

		// The middleware wraps the body in someCustomBody, which net/http
		// does not recognize. GetBody stays nil, so after the GOAWAY net/http2
		// cannot replay the request and returns an error.
		_, err := outboundWithMiddleware.Call(ctx, &transport.Request{
			Caller:    "caller",
			Service:   "service",
			Encoding:  raw.Encoding,
			Procedure: "hello",
			Body:      strings.NewReader("world"),
		})
		require.Error(t, err, "expected replay to fail when middleware wraps the body in a non-rewindable type")
		assert.Contains(t, err.Error(), "GetBody", "error should mention GetBody")
	})
}
