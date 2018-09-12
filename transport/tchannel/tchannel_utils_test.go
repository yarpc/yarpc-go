// Copyright (c) 2018 Uber Technologies, Inc.
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

package tchannel

import (
	"bytes"
	"errors"
	"io/ioutil"

	"github.com/uber/tchannel-go"
)

func readArgs(r tchannel.ArgReadable) (arg2, arg3 []byte, err error) {
	err = tchannel.NewArgReader(r.Arg2Reader()).Read(&arg2)
	if err != nil {
		return
	}

	err = tchannel.NewArgReader(r.Arg3Reader()).Read(&arg3)
	return
}

func writeArgs(w tchannel.ArgWritable, arg2, arg3 []byte) error {
	if err := tchannel.NewArgWriter(w.Arg2Writer()).Write(arg2); err != nil {
		return err
	}

	return tchannel.NewArgWriter(w.Arg3Writer()).Write(arg3)
}

// This file provides utilities to help test TChannel behavior used by
// multiple tests.

// bufferArgWriter is a Buffer that satisfies the tchannel.ArgWriter
// interface.
type bufferArgWriter struct{ bytes.Buffer }

func newBufferArgWriter() *bufferArgWriter {
	return new(bufferArgWriter)
}

func (w *bufferArgWriter) Close() error { return nil }
func (w *bufferArgWriter) Flush() error { return nil }

// fakeInboundCall is a fake inboundCall that uses a responseRecorder to
// record responses.
//
// Provide nil for arg2 or arg3 to get Arg2Reader or Arg3Reader to fail.
type fakeInboundCall struct {
	service         string
	caller          string
	method          string
	shardkey        string
	routingkey      string
	routingdelegate string
	format          tchannel.Format
	arg2, arg3      []byte
	resp            inboundCallResponse
}

func (i *fakeInboundCall) ServiceName() string           { return i.service }
func (i *fakeInboundCall) CallerName() string            { return i.caller }
func (i *fakeInboundCall) MethodString() string          { return i.method }
func (i *fakeInboundCall) ShardKey() string              { return i.shardkey }
func (i *fakeInboundCall) RoutingKey() string            { return i.routingkey }
func (i *fakeInboundCall) RoutingDelegate() string       { return i.routingdelegate }
func (i *fakeInboundCall) Format() tchannel.Format       { return i.format }
func (i *fakeInboundCall) Response() inboundCallResponse { return i.resp }

func (i *fakeInboundCall) Arg2Reader() (tchannel.ArgReader, error) {
	if i.arg2 == nil {
		return nil, errors.New("no arg2 provided")
	}
	return ioutil.NopCloser(bytes.NewReader(i.arg2)), nil
}

func (i *fakeInboundCall) Arg3Reader() (tchannel.ArgReader, error) {
	if i.arg3 == nil {
		return nil, errors.New("no arg3 provided")
	}
	return ioutil.NopCloser(bytes.NewReader(i.arg3)), nil
}

// responseRecorder is a inboundCallResponse that records whatever is written
// to it.
//
// The recorder will throw an error if arg2 or arg3 are set to nil.
type responseRecorder struct {
	arg2, arg3       *bufferArgWriter
	systemErr        error
	applicationError bool
	blackholed       bool
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{
		arg2: newBufferArgWriter(),
		arg3: newBufferArgWriter(),
	}
}

func (rr *responseRecorder) Arg2Writer() (tchannel.ArgWriter, error) {
	if rr.arg2 == nil {
		return nil, errors.New("no arg2 provided")
	}
	return rr.arg2, nil
}

func (rr *responseRecorder) Arg3Writer() (tchannel.ArgWriter, error) {
	if rr.arg3 == nil {
		return nil, errors.New("no arg3 provided")
	}
	return rr.arg3, nil
}

func (rr *responseRecorder) SendSystemError(err error) error {
	rr.systemErr = err
	return nil
}

func (rr *responseRecorder) SetApplicationError() error {
	rr.applicationError = true
	return nil
}

func (rr *responseRecorder) Blackhole() {
	rr.blackholed = true
}

// faultyResponseWriter mocks a responseWriter.Close() error to test logging behaviour
// inside tchannel.Handle.
type faultyResponseWriter struct{ tchannelResponseWriter }

func (frw *faultyResponseWriter) Close() error {
	return errors.New("faultyResponseWriter failed to close")
}

func newFaultyResponseWriter(response inboundCallResponse, format tchannel.Format, headerCase headerCase) responseWriter {
	return &faultyResponseWriter{
		tchannelResponseWriter{
			Response:   response,
			Format:     format,
			HeaderCase: headerCase,
		},
	}
}
