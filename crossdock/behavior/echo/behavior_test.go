// Copyright (c) 2016 Uber Technologies, Inc.
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

package echo

import (
	"testing"

	"github.com/yarpc/yarpc-go/crossdock-go"

	"github.com/stretchr/testify/assert"
)

func TestEchoSink(t *testing.T) {
	tests := []struct {
		server, transport, encoding string
		act                         func(crossdock.Sink)
		entry                       echoEntry
	}{
		{
			"localhost", "tchannel", "json",
			func(s crossdock.Sink) {
				crossdock.Successf(s, "it worked!")
			},
			echoEntry{
				Entry: crossdock.Entry{
					Status: crossdock.Passed,
					Output: "it worked!",
				},
				Transport: "tchannel",
				Encoding:  "json",
				Server:    "localhost",
			},
		},
		{
			"localhost", "http", "thrift",
			func(s crossdock.Sink) {
				crossdock.Skipf(s, "what even is")
			},
			echoEntry{
				Entry: crossdock.Entry{
					Status: crossdock.Skipped,
					Output: "what even is",
				},
				Transport: "http",
				Encoding:  "thrift",
				Server:    "localhost",
			},
		},
		{
			"localhost", "http", "raw",
			func(s crossdock.Sink) {
				crossdock.Fatalf(s, "great sadness")
			},
			echoEntry{
				Entry: crossdock.Entry{
					Status: crossdock.Failed,
					Output: "great sadness",
				},
				Transport: "http",
				Encoding:  "raw",
				Server:    "localhost",
			},
		},
	}

	for _, tt := range tests {
		entries := crossdock.Run(func(s crossdock.Sink) {
			es := createEchoSink(tt.encoding, s, crossdock.ParamsFromMap{
				"server":    tt.server,
				"transport": tt.transport,
			})
			tt.act(es)
		})

		if assert.Len(t, entries, 1) {
			assert.Equal(t, tt.entry, entries[0])
		}
	}
}
