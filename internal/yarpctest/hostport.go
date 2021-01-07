// Copyright (c) 2021 Uber Technologies, Inc.
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

package yarpctest

import (
	"net"
	"strings"
)

const localhostString = "127.0.0.1"

// ZeroAddrToHostPort converts a net.Addr created with net.Listen("tcp", ":0")
// to a hostPort string valid to use for Golang versions <= 1.8
//
// See the "net" section in https://golang.org/doc/go1.9#minor_library_changes
// for more details.
func ZeroAddrToHostPort(addr net.Addr) string {
	return ZeroAddrStringToHostPort(addr.String())
}

// ZeroAddrStringToHostPort converts a string from net.Addr.String() created
// with net.Listen("tcp", ":0") to a hostPort string valid to use for Golang
// versions <= 1.8
//
// See the "net" section in https://golang.org/doc/go1.9#minor_library_changes
// for more details.
func ZeroAddrStringToHostPort(addrString string) string {
	return localhostString + ":" + ZeroAddrStringToPort(addrString)
}

// ZeroAddrToPort converts a net.Addr created with net.Listen("tcp", ":0")
// to a port string valid to use for Golang versions <= 1.8
//
// See the "net" section in https://golang.org/doc/go1.9#minor_library_changes
// for more details.
func ZeroAddrToPort(addr net.Addr) string {
	return ZeroAddrStringToPort(addr.String())
}

// ZeroAddrStringToPort converts a string from net.Addr.String() created
// with net.Listen("tcp", ":0") to a port string valid to use for Golang
// versions <= 1.8
//
// See the "net" section in https://golang.org/doc/go1.9#minor_library_changes
// for more details.
func ZeroAddrStringToPort(addrString string) string {
	split := strings.Split(addrString, ":")
	return split[len(split)-1]
}
