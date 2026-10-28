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
	"bufio"
	"net"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// newBufferedH2CHandler keeps a read buffer on hijacked cleartext HTTP/2
// connections. h2c's own buffer only drains bytes left by the HTTP/1 parser;
// after that, the HTTP/2 framer reads frame headers and payloads separately
// from the socket. A persistent buffer combines those small reads.
func newBufferedH2CHandler(handler http.Handler, server *http2.Server) http.Handler {
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ordinary HTTP/1 handlers must retain the original writer's optional
		// interfaces, including Flusher and Hijacker.
		if buffered, ok := w.(*h2cReadBufferWriter); ok {
			w = buffered.ResponseWriter
		}
		handler.ServeHTTP(w, r)
	})
	h2cHandler := h2c.NewHandler(fallback, server)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Leave TLS connections and their connection state untouched. TLS
		// already buffers records; this buffer targets cleartext framing.
		// Ordinary HTTP/1 requests do not need a writer wrapper either.
		if r.TLS == nil && (r.Method == "PRI" || len(r.Header["Upgrade"]) > 0) {
			if hijacker, ok := w.(http.Hijacker); ok {
				w = &h2cReadBufferWriter{ResponseWriter: w, Hijacker: hijacker}
			}
		}
		h2cHandler.ServeHTTP(w, r)
	})
}

type h2cReadBufferWriter struct {
	http.ResponseWriter
	http.Hijacker
}

func (w *h2cReadBufferWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := w.Hijacker.Hijack()
	if err != nil {
		return conn, rw, err
	}
	// Preserve rw: h2c must consume the HTTP/1 parser's buffered bytes before
	// switching to conn. bufio.NewReader uses a 4 KiB buffer per connection.
	return &h2cReadBufferConn{Conn: conn, reader: bufio.NewReader(conn)}, rw, nil
}

type h2cReadBufferConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *h2cReadBufferConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}
