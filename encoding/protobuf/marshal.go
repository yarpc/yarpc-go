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

package protobuf

import (
	"bytes"
	"io"
	"sync"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
)

var (
	_bufferPool = sync.Pool{
		New: func() interface{} {
			return proto.NewBuffer(make([]byte, 1024))
		},
	}
)

// codec is a private helper struct used to hold custom marshaling behavior
// for JSON encoding (specifically, the AnyResolver).
type codec struct {
	jsonMarshaler   *jsonpb.Marshaler
	jsonUnmarshaler *jsonpb.Unmarshaler
}

func newCodec(anyResolver jsonpb.AnyResolver) *codec {
	return &codec{
		jsonMarshaler:   &jsonpb.Marshaler{AnyResolver: anyResolver},
		jsonUnmarshaler: &jsonpb.Unmarshaler{AnyResolver: anyResolver, AllowUnknownFields: true},
	}
}

func unmarshal(encoding transport.Encoding, reader io.Reader, message proto.Message, c *codec) error {
	buf := bufferpool.Get()
	defer bufferpool.Put(buf)
	if _, err := buf.ReadFrom(reader); err != nil {
		return err
	}
	body := buf.Bytes()
	if len(body) == 0 {
		return nil
	}
	return unmarshalBytes(encoding, body, message, c)
}

func unmarshalBytes(encoding transport.Encoding, body []byte, message proto.Message, c *codec) error {
	customCodec := getCodec(encoding)
	if customCodec == nil {
		return newUnrecognizedEncodingError(encoding)
	}
	// For JSON encoding with custom AnyResolver, use the provided codec
	if encoding == JSONEncoding && c != nil {
		return c.jsonUnmarshaler.Unmarshal(bytes.NewReader(body), message)
	}
	return customCodec.Unmarshal(body, message)
}

func marshal(encoding transport.Encoding, message proto.Message, c *codec) ([]byte, func(), error) {
	customCodec := getCodec(encoding)
	if customCodec == nil {
		return nil, func() {}, newUnrecognizedEncodingError(encoding)
	}
	// For JSON encoding with custom AnyResolver, use the provided codec
	if encoding == JSONEncoding && c != nil {
		buf := bufferpool.Get()
		cleanup := func() { bufferpool.Put(buf) }
		if err := c.jsonMarshaler.Marshal(buf, message); err != nil {
			cleanup()
			return nil, func() {}, err
		}
		return buf.Bytes(), cleanup, nil
	}
	return customCodec.Marshal(message)
}

func getBuffer() *proto.Buffer {
	buf := _bufferPool.Get().(*proto.Buffer)
	buf.Reset()
	return buf
}

func putBuffer(buf *proto.Buffer) {
	_bufferPool.Put(buf)
}
