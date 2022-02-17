// Copyright (c) 2022 Uber Technologies, Inc.
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

package v2

import (
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"io"
	"sync"
)

var (
	_bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 1024)
		},
	}
)

const (
	// Encoding is the name of this encoding.
	Encoding transport.Encoding = "proto"

	// JSONEncoding is the name of the JSON encoding.
	// Protobuf handlers are able to handle both Encoding and JSONEncoding encodings.
	JSONEncoding transport.Encoding = "json"
)

// AnyResolver provides interface for looking up or iterating over descriptor types.
type AnyResolver interface {
	protoregistry.ExtensionTypeResolver
	protoregistry.MessageTypeResolver
}

// codec is a private helper struct used to hold custom marshling behavior of golang protobuf messages.
type codec struct {
	jsonMarshaler   *protojson.MarshalOptions
	jsonUnmarshaler *protojson.UnmarshalOptions
}

func newCodec(anyResolver AnyResolver) *codec {
	return &codec{
		jsonMarshaler:   &protojson.MarshalOptions{Resolver: anyResolver},
		jsonUnmarshaler: &protojson.UnmarshalOptions{Resolver: anyResolver},
	}
}

func unmarshal(encoding transport.Encoding, reader io.Reader, message proto.Message, codec *codec) error {
	buf := bufferpool.Get()
	defer bufferpool.Put(buf)
	if _, err := buf.ReadFrom(reader); err != nil {
		return err
	}
	body := buf.Bytes()
	if len(body) == 0 {
		return nil
	}
	return unmarshalBytes(encoding, body, message, codec)
}

func unmarshalBytes(encoding transport.Encoding, body []byte, message proto.Message, codec *codec) error {
	switch encoding {
	case Encoding:
		return unmarshalProto(body, message, codec)
	case JSONEncoding:
		return unmarshalJSON(body, message, codec)
	default:
		return yarpcerrors.Newf(yarpcerrors.CodeInternal, "encoding.Expect should have handled encoding %q but did not", encoding)
	}
}

func unmarshalProto(body []byte, message proto.Message, _ *codec) error {
	return proto.Unmarshal(body, message)
}

func unmarshalJSON(body []byte, message proto.Message, codec *codec) error {
	return codec.jsonUnmarshaler.Unmarshal(body, message)
}

func marshal(encoding transport.Encoding, message proto.Message, codec *codec) ([]byte, func(), error) {
	switch encoding {
	case Encoding:
		return marshalProto(message, codec)
	case JSONEncoding:
		return marshalJSON(message, codec)
	default:
		return nil, nil, yarpcerrors.Newf(yarpcerrors.CodeInternal, "encoding.Expect should have handled encoding %q but did not", encoding)
	}
}

func marshalProto(message proto.Message, _ *codec) ([]byte, func(), error) {
	buf := getBuffer()
	cleanup := func() { putBuffer(buf) }
	data, err := proto.MarshalOptions{}.MarshalAppend(buf, message)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return data, func() { putBuffer(data) }, nil
}

func marshalJSON(message proto.Message, codec *codec) ([]byte, func(), error) {
	data, err := codec.jsonMarshaler.Marshal(message)
	if err != nil {
		return nil, nil, err
	}
	return data, func() {}, nil
}

func getBuffer() []byte {
	buf := _bufferPool.Get().([]byte)
	buf = buf[:0]
	return buf
}

func putBuffer(buf []byte) {
	_bufferPool.Put(buf)
}
