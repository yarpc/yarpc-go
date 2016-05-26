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
	"github.com/yarpc/yarpc-go/crossdock-go/crossdock"
	"github.com/yarpc/yarpc-go/crossdock/client/params"
)

// echoEntry is an entry emitted by the echo behaviors.
type echoEntry struct {
	crossdock.Entry

	Transport string `json:"transport"`
	Encoding  string `json:"encoding"`
	Server    string `json:"server"`
}

// echoSink wraps a sink to emit echoEntry entries instead.
type echoSink struct {
	crossdock.Sink

	Transport string
	Encoding  string
	Server    string
}

func (s echoSink) Put(e interface{}) {
	s.Sink.Put(echoEntry{
		Entry:     e.(crossdock.Entry),
		Transport: s.Transport,
		Encoding:  s.Encoding,
		Server:    s.Server,
	})
}

// createEchoSink wraps a Sink to have transport, encoding, and server
// information.
func createEchoSink(encoding string, s crossdock.Sink, p crossdock.Params) crossdock.Sink {
	return echoSink{
		Sink:      s,
		Transport: p.Param(params.Transport),
		Encoding:  encoding,
		Server:    p.Param(params.Server),
	}
}
