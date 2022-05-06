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

import "io"

const (
	_tlsHandshakeHeaderLength = 5

	// Based on handshake(22) of ContentType from:
	// https://datatracker.ietf.org/doc/html/rfc8446#section-5.1
	_tlsContentTypeHandshake = 22
	// TLS 1.0 is rename of SSL3.1, which implies major version is 3 and minor
	// version is >= 1.
	_tlsMajorVersion = 3
	_tlsMinorVersion = 1

	// Offsets below have been derived from TLSPlaintext struct from:
	// https://datatracker.ietf.org/doc/html/rfc8446#section-5.1
	_tlsContentTypeOffset  = 0
	_tlsMajorVersionOffset = 1
	_tlsMinorVersionOffset = 2
)

// isTLSClientHelloRecord returns true when the reader contains TLS client hello
// record in the initial bytes.
// Read more about header spec: https://datatracker.ietf.org/doc/html/rfc8446#section-5.1
func isTLSClientHelloRecord(r io.Reader) (bool, error) {
	buf := make([]byte, _tlsHandshakeHeaderLength)
	n, err := r.Read(buf)
	if err != nil {
		return false, err
	}

	if n != _tlsHandshakeHeaderLength {
		return false, nil
	}

	return buf[_tlsContentTypeOffset] == _tlsContentTypeHandshake &&
		buf[_tlsMajorVersionOffset] == _tlsMajorVersion &&
		buf[_tlsMinorVersionOffset] >= _tlsMinorVersion, nil
}
