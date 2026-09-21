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

// Package peeraddr formats and parses the indexed peer identifiers that let a
// peer list name the same network address more than once.
//
// A configured address that appears N times becomes N distinct identifiers,
// "host:port#1" through "host:port#N". Transports key their peer maps on the
// full identifier but dial Address(identifier), the bare "host:port".
package peeraddr

import (
	"strconv"
	"strings"
)

// separator sits between the network address and the occurrence index.
const separator = "#"

// Indexed returns the identifier for the n-th occurrence of addr in a peer
// list, for example Indexed("127.0.0.1:8080", 2) == "127.0.0.1:8080#2".
func Indexed(addr string, n int) string {
	return addr + separator + strconv.Itoa(n)
}

// Address strips the occurrence index from an identifier produced by Indexed,
// returning the dialable network address. Identifiers without an index are
// returned unchanged.
func Address(identifier string) string {
	if i := strings.LastIndex(identifier, separator); i >= 0 {
		return identifier[:i]
	}
	return identifier
}
