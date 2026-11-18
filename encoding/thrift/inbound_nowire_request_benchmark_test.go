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

package thrift

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/thriftrw/protocol/binary"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/wire"
)

// BenchmarkNoWireRequestDecode is the inbound counterpart of
// BenchmarkNoWireResponseDecode: the server decodes a request made of many
// short Thrift strings, which costs a length read and a payload read each.
// The TChannel case controls for requests the transport already holds in
// memory and which are therefore not wrapped.
func BenchmarkNoWireRequestDecode(b *testing.B) {
	for _, transportName := range []string{"http2", "tchannel"} {
		b.Run(transportName, func(b *testing.B) {
			for _, shape := range []struct {
				name  string
				count int
			}{
				{name: "small", count: 8},
				{name: "large", count: 1280},
			} {
				b.Run(shape.name, func(b *testing.B) {
					values := make([]string, shape.count)
					for i := range values {
						values[i] = strings.Repeat("x", 96)
					}
					client := newNoWireBenchmarkClient(b, transportName, thriftNoWireHandler{
						Handler:       requestBenchmarkHandler{want: values},
						RequestReader: binary.Default,
					}, Enveloped)
					request := stringListRequest(values)
					b.ReportAllocs()
					b.SetBytes(int64(len(encodeStringList(b, values))))
					b.ResetTimer()
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
							err := client.Call(ctx, request, responseDecoderFunc(func(sr stream.Reader) error {
								_, err := sr.ReadString()
								return err
							}))
							cancel()
							if err != nil {
								b.Error(err)
								return
							}
						}
					})
				})
			}
		})
	}
}

// requestBenchmarkHandler decodes the request list and verifies it, as a
// generated handler would decode its arguments.
type requestBenchmarkHandler struct {
	want []string
}

func (h requestBenchmarkHandler) HandleNoWire(ctx context.Context, nwc *NoWireCall) (NoWireResponse, error) {
	var got stringListResponse
	rw, err := nwc.RequestReader.ReadRequest(ctx, nwc.EnvelopeType, nwc.Reader, &got)
	if err != nil {
		return NoWireResponse{}, err
	}
	if len(got) != len(h.want) || got[len(got)-1] != h.want[len(h.want)-1] {
		return NoWireResponse{}, errBenchmarkRequest
	}
	return NoWireResponse{Body: fakeEnveloper(wire.Reply), ResponseWriter: rw}, nil
}

type benchmarkError string

func (e benchmarkError) Error() string { return string(e) }

const errBenchmarkRequest = benchmarkError("incorrect request")

// stringListRequest is a request Enveloper that writes a list of strings.
type stringListRequest []string

func (stringListRequest) MethodName() string { return "someMethod" }

func (stringListRequest) EnvelopeType() wire.EnvelopeType { return wire.Call }

func (r stringListRequest) Encode(sw stream.Writer) error {
	if err := sw.WriteListBegin(stream.ListHeader{Type: wire.TBinary, Length: len(r)}); err != nil {
		return err
	}
	for _, v := range r {
		if err := sw.WriteString(v); err != nil {
			return err
		}
	}
	return sw.WriteListEnd()
}
