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
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

func TestBufferedH2CRoundTrip(t *testing.T) {
	for _, tt := range []struct {
		name    string
		http2   bool
		tls     bool
		upgrade bool
	}{
		{name: "HTTP1"},
		{name: "HTTP1 upgrade fallback", upgrade: true},
		{name: "prior knowledge", http2: true},
		{name: "TLS HTTP1", tls: true},
		{name: "TLS HTTP2", tls: true, http2: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, flusher := w.(http.Flusher)
				assert.True(t, flusher, "fallback must preserve optional writer interfaces")
				if r.ProtoMajor == 1 {
					_, hijacker := w.(http.Hijacker)
					assert.True(t, hijacker)
				}
				w.Header().Set("X-TLS", strconv.FormatBool(r.TLS != nil))
				data, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				_, err = w.Write(data)
				assert.NoError(t, err)
			})
			server := httptest.NewUnstartedServer(newBufferedH2CHandler(handler, &http2.Server{}))
			server.EnableHTTP2 = tt.http2
			if tt.tls {
				server.StartTLS()
			} else {
				server.Start()
			}
			t.Cleanup(server.Close)
			client := server.Client()
			if tt.http2 && !tt.tls {
				client.Transport = &http2.Transport{
					AllowHTTP: true,
					DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
						var dialer net.Dialer
						return dialer.DialContext(ctx, network, addr)
					},
				}
			}
			client.Timeout = 5 * time.Second
			t.Cleanup(client.CloseIdleConnections)

			// Reuse the connection with bodies on either side of the buffer
			// size. Leftover buffered bytes must not corrupt the next request.
			for index, size := range []int{31, 128 * 1024, 17} {
				body := strings.Repeat(strconv.Itoa(index), size)
				request, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(body))
				require.NoError(t, err)
				if tt.upgrade {
					request.Header.Set("Upgrade", "websocket")
					request.Header.Set("Connection", "Upgrade")
				}
				response, err := client.Do(request)
				require.NoError(t, err)
				data, readErr := io.ReadAll(response.Body)
				require.NoError(t, response.Body.Close())
				require.NoError(t, readErr)
				assert.Equal(t, http.StatusOK, response.StatusCode)
				assert.Len(t, data, len(body))
				assert.True(t, bytes.Equal([]byte(body), data), "response must echo the complete request body")
				assert.Equal(t, strconv.FormatBool(tt.tls), response.Header.Get("X-TLS"))
				wantProto := 1
				if tt.http2 {
					wantProto = 2
				}
				assert.Equal(t, wantProto, response.ProtoMajor)
			}
		})
	}
}

func TestBufferedH2CUpgrade(t *testing.T) {
	// Send the upgrade body, client preface and a PING in a single write.
	// Hijack's reader may already contain bytes from the HTTP/2 connection;
	// replacing that reader would lose them.
	body := "body received before hijacking"
	server := httptest.NewServer(newBufferedH2CHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.Copy(w, r.Body)
		assert.NoError(t, err)
	}), &http2.Server{}))
	t.Cleanup(server.Close)
	conn, err := net.Dial("tcp", strings.TrimPrefix(server.URL, "http://"))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, conn.Close()) })
	require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))

	var request bytes.Buffer
	fmt.Fprintf(&request, "POST / HTTP/1.1\r\nHost: localhost\r\nConnection: Upgrade, HTTP2-Settings\r\nUpgrade: h2c\r\nHTTP2-Settings: \r\nContent-Length: %d\r\n\r\n%s", len(body), body)
	request.WriteString(http2.ClientPreface)
	frames := http2.NewFramer(&request, nil)
	require.NoError(t, frames.WriteSettings())
	ping := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	require.NoError(t, frames.WritePing(false, ping))
	_, err = conn.Write(request.Bytes())
	require.NoError(t, err)

	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodPost})
	require.NoError(t, err)
	assert.Equal(t, http.StatusSwitchingProtocols, response.StatusCode)
	framer := http2.NewFramer(conn, reader)
	var received bytes.Buffer
	var streamEnded, pingAck bool
	for !streamEnded || !pingAck {
		frame, err := framer.ReadFrame()
		require.NoError(t, err)
		switch frame := frame.(type) {
		case *http2.SettingsFrame:
			if !frame.IsAck() {
				require.NoError(t, framer.WriteSettingsAck())
			}
		case *http2.DataFrame:
			require.EqualValues(t, 1, frame.StreamID)
			received.Write(frame.Data())
			streamEnded = frame.StreamEnded()
		case *http2.PingFrame:
			if frame.IsAck() {
				assert.Equal(t, ping, frame.Data)
				pingAck = true
			}
		}
	}
	assert.Equal(t, body, received.String())
}
