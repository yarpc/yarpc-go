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

package tchannel

import (
	"encoding/binary"
	"io"

	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/tchannel/internal"

	"github.com/uber/tchannel-go"
)

// readHeaders reads headers using the given function to get the arg reader.
//
// This may be used with the Arg2Reader functions on InboundCall and
// OutboundCallResponse.
//
// If the format is JSON, the headers are expected to be JSON encoded.
func readHeaders(format tchannel.Format, getReader func() (tchannel.ArgReader, error)) (transport.Headers, error) {
	if format == tchannel.JSON {
		// JSON is special
		var headers transport.Headers
		err := tchannel.NewArgReader(getReader()).ReadJSON(&headers)
		return headers, err
	}

	r, err := getReader()
	if err != nil {
		return nil, err
	}

	headers, err := decodeHeaders(r)
	if err != nil {
		return nil, err
	}

	return headers, r.Close()
}

// writeHeaders writes the given headers using the given function to get the
// arg writer.
//
// This may be used with the Arg2Writer functions on OutboundCall and
// InboundCallResponse.
//
// If the format is JSON, the headers are JSON encoded.
func writeHeaders(format tchannel.Format, headers transport.Headers, getWriter func() (tchannel.ArgWriter, error)) error {
	if format == tchannel.JSON {
		// JSON is special
		return tchannel.NewArgWriter(getWriter()).WriteJSON(headers)
	}
	return tchannel.NewArgWriter(getWriter()).Write(encodeHeaders(headers))
}

// decodeHeaders decodes headers using the format:
//
// 	nh:2 (k~2 v~2){nh}
func decodeHeaders(r io.Reader) (transport.Headers, error) {
	reader := internal.NewReader(r)

	count := reader.ReadUint16()
	if count == 0 {
		return nil, reader.Err()
	}

	headers := make(transport.Headers, count)
	for i := 0; i < int(count) && reader.Err() == nil; i++ {
		k := reader.ReadLen16String()
		v := reader.ReadLen16String()
		headers[k] = v
	}

	return headers, reader.Err()
}

// encodeHeaders encodes headers using the format:
//
// 	nh:2 (k~2 v~2){nh}
func encodeHeaders(hs transport.Headers) []byte {
	if hs == nil || len(hs) == 0 {
		return []byte{0, 0} // nh = 2
	}

	size := 2 // nh:2
	for k, v := range hs {
		size += len(k) + 2 // k~2
		size += len(v) + 2 // v~2
	}

	out := make([]byte, size)

	i := 2
	binary.BigEndian.PutUint16(out, uint16(len(hs))) // nh:2
	for k, v := range hs {
		i += _putStr16(k, out[i:]) // k~2
		i += _putStr16(v, out[i:]) // v~2
	}

	return out
}

// _putStr16 writes the bytes `in` into `out` using the encoding `s~2`.
func _putStr16(in string, out []byte) int {
	binary.BigEndian.PutUint16(out, uint16(len(in)))
	return copy(out[2:], in) + 2
}
