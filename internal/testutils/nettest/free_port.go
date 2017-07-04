// Copyright (c) 2017 Uber Technologies, Inc.
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

// Package nettest provides helpers to get free host:ports for tests that
// require fixed host:ports and cannot listen on port 0.
package nettest

import "net"

func getClosedTCPAddr() (*net.TCPAddr, error) {
	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return nil, err
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr), nil
}

// MustGetFreeHostPort returns a TCP host:port that is free for unit tests
// that cannot use port 0.
func MustGetFreeHostPort() string {
	addr, err := getClosedTCPAddr()
	if err != nil {
		panic(err)
	}
	return addr.String()
}

// MustGetFreePort returns a TCP port that is free for unit tests that cannot
// use port 0.
func MustGetFreePort() uint16 {
	addr, err := getClosedTCPAddr()
	if err != nil {
		panic(err)
	}
	return uint16(addr.Port)
}
