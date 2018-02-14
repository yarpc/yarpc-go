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
	"context"
	"encoding/binary"
	"io"
	"strings"

	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/tchannel/internal"
)

const (
	// ErrorCodeHeaderKey is the response header key for the error code.
	ErrorCodeHeaderKey = "$rpc$-error-code"
	// ErrorNameHeaderKey is the response header key for the error name.
	ErrorNameHeaderKey = "$rpc$-error-name"
	// ErrorMessageHeaderKey is the response header key for the error message.
	ErrorMessageHeaderKey = "$rpc$-error-message"
)

var _reservedHeaderKeys = map[string]bool{
	ErrorCodeHeaderKey:    true,
	ErrorNameHeaderKey:    true,
	ErrorMessageHeaderKey: true,
}

func isReservedHeaderKey(key string) bool {
	_, ok := _reservedHeaderKeys[strings.ToLower(key)]
	return ok
}

// readRequestHeaders reads headers and baggage from an incoming request.
func readRequestHeaders(
	ctx context.Context,
	format tchannel.Format,
	getReader func() (tchannel.ArgReader, error),
) (context.Context, transport.Headers, error) {
	headers, err := readHeaders(format, getReader)
	if err != nil {
		return ctx, headers, err
	}
	return ctx, headers, nil
}

// readHeaders reads headers using the given function to get the arg reader.
//
// This may be used with the Arg2Reader functions on InboundCall and
// OutboundCallResponse.
//
// If the format is JSON, the headers are expected to be JSON encoded.
//
// This function always returns a non-nil Headers object in case of success.
func readHeaders(format tchannel.Format, getReader func() (tchannel.ArgReader, error)) (transport.Headers, error) {
	if format == tchannel.JSON {
		// JSON is special
		var headers map[string]string
		err := tchannel.NewArgReader(getReader()).ReadJSON(&headers)
		return transport.HeadersFromMap(headers), err
	}

	r, err := getReader()
	if err != nil {
		return transport.Headers{}, err
	}

	headers, err := decodeHeaders(r)
	if err != nil {
		return headers, err
	}

	return headers, r.Close()
}

func writeRequestHeaders(
	ctx context.Context,
	format tchannel.Format,
	appHeaders map[string]string,
	getWriter func() (tchannel.ArgWriter, error),
) error {
	headers := transport.NewHeadersWithCapacity(len(appHeaders))
	// TODO: zero-alloc version

	for k, v := range appHeaders {
		headers = headers.With(k, v)
	}

	return writeHeaders(format, headers, getWriter)
}

var emptyMap = map[string]string{}

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
		items := headers.Items()
		if items == nil {
			// We want to write "{}", not "null" for empty map.
			items = emptyMap
		}
		return tchannel.NewArgWriter(getWriter()).WriteJSON(items)
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
		return transport.Headers{}, reader.Err()
	}

	headers := transport.NewHeadersWithCapacity(int(count))
	for i := 0; i < int(count) && reader.Err() == nil; i++ {
		k := reader.ReadLen16String()
		v := reader.ReadLen16String()
		headers = headers.With(k, v)
	}

	return headers, reader.Err()
}

// encodeHeaders encodes headers using the format:
//
// 	nh:2 (k~2 v~2){nh}
func encodeHeaders(hs transport.Headers) []byte {
	if hs.Len() == 0 {
		return []byte{0, 0} // nh = 2
	}

	size := 2 // nh:2
	for k, v := range hs.Items() {
		size += len(k) + 2 // k~2
		size += len(v) + 2 // v~2
	}

	out := make([]byte, size)

	i := 2
	binary.BigEndian.PutUint16(out, uint16(hs.Len())) // nh:2
	for k, v := range hs.Items() {
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
