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

package peeraddr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexed(t *testing.T) {
	assert.Equal(t, "127.0.0.1:8080#1", Indexed("127.0.0.1:8080", 1))
	assert.Equal(t, "127.0.0.1:8080#12", Indexed("127.0.0.1:8080", 12))
	assert.Equal(t, "host.example.com:80#2", Indexed("host.example.com:80", 2))
}

func TestAddress(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{name: "no index", identifier: "127.0.0.1:8080", want: "127.0.0.1:8080"},
		{name: "single digit", identifier: "127.0.0.1:8080#1", want: "127.0.0.1:8080"},
		{name: "multi digit", identifier: "127.0.0.1:8080#12", want: "127.0.0.1:8080"},
		{name: "only last separator is stripped", identifier: "127.0.0.1:8080#1#2", want: "127.0.0.1:8080#1"},
		{name: "hostname", identifier: "host.example.com:80#3", want: "host.example.com:80"},
		{name: "empty", identifier: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Address(tt.identifier))
		})
	}
}

func TestAddressRoundTripsIndexed(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:8080", "[::1]:443", "host:1"} {
		for n := 1; n <= 3; n++ {
			assert.Equal(t, addr, Address(Indexed(addr, n)))
		}
	}
}
