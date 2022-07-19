// Copyright (c) 2022 Uber Technologies, Inc.
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

package tlsmux

import (
	"bytes"
	"net"
)

// connSniffer wraps the connection and enables tlsmux to sniff inital bytes from the
// connection efficiently.
type connSniffer struct {
	net.Conn

	// set to true when sniffing mode is disabled.
	disableSniffing bool
	// buf stores bytes read from the underlying connection when in sniffing
	// mode. When sniffing mode is disabled, buffered bytes is returned.
	buf bytes.Buffer
}

func newConnectionSniffer(conn net.Conn) *connSniffer {
	return &connSniffer{Conn: conn}
}

// Read returns bytes read from the underlying connection. When sniffing is
// true, data read from the connection is stored in the buffer. When sniffing
// mode is disabled, data is first read from the buffer and once the buffer is
// empty the underlying connection is read.
func (c *connSniffer) Read(b []byte) (int, error) {
	if c.disableSniffing && c.buf.Len() != 0 {
		// Read from the buffer when sniffing is disabled and buffer is not empty.
		n, _ := c.buf.Read(b)
		if c.buf.Len() == 0 {
			// Release memory as we don't need buffer anymore.
			c.buf = bytes.Buffer{}
		}
		return n, nil
	}

	n, err := c.Conn.Read(b)
	if err != nil {
		return n, err
	}

	// Store in buffer when sniffing.
	if !c.disableSniffing {
		c.buf.Write(b[:n])
	}
	return n, nil
}

func (c *connSniffer) stopSniffing() {
	c.disableSniffing = true
}
