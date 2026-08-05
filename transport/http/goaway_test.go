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
		mw      middleware.UnaryOutbound
		reqBody io.Reader
		wantErr bool
	}{
		{
			name:    "no-op middleware, valid Request.Body; relay occurs",
			mw:      middleware.NopUnaryOutbound,
			reqBody: getH2ValidBody(),
			wantErr: false,
		},
		{
			name:    "no-op middleware, invalid Request.Body; relay fails",
			mw:      middleware.NopUnaryOutbound,
			reqBody: getH2InvalidBody(),
			wantErr: true,
		},
		{
			name:    "custom middleware, valid Request.Body becomes invalid; replay fails",
			mw:      someCustomMiddleware{},
			reqBody: getH2ValidBody(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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

			outboundWithMiddleware := middleware.ApplyUnaryOutbound(out, tt.mw)

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

func getH2ValidBody() io.Reader {
	// transport.Request.Body must be one of *bytes.Buffer, *bytes.Reader, or *strings.Reader.
	return strings.NewReader("some-payload")
}

func getH2InvalidBody() io.Reader {
	// Any other transport.Request.Body concrete type that does not allow automatic GetBody creation.
	return &someCustomBody{reader: strings.NewReader("some-payload")}
}

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
