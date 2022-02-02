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
	"bytes"
	"io"
	"sync"

	"github.com/golang/protobuf/jsonpb"
	golang_proto "github.com/golang/protobuf/proto"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/protobuf/proto"
)

const (
	// Encoding is the name of this encoding.
	Encoding transport.Encoding = "proto"

	// JSONEncoding is the name of the JSON encoding.
	// Protobuf handlers are able to handle both Encoding and JSONEncoding encodings.
	JSONEncoding transport.Encoding = "json"
)

var (
	_bufferPool = sync.Pool{
		New: func() interface{} {
			return golang_proto.NewBuffer(make([]byte, 1024))
		},
	}
)

// codec is a private helper struct used to hold custom marshling behavior of golang protobuf messages.
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
	return codec.jsonUnmarshaler.Unmarshal(bytes.NewReader(body), golang_proto.MessageV1(message))
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
	protoBuffer := getBuffer()
	cleanup := func() { putBuffer(protoBuffer) }
	if err := protoBuffer.Marshal(golang_proto.MessageV1(message)); err != nil {
		cleanup()
		return nil, nil, err
	}
	return protoBuffer.Bytes(), cleanup, nil
}

func getBuffer() *golang_proto.Buffer {
	buf := _bufferPool.Get().(*golang_proto.Buffer)
	buf.Reset()
	return buf
}

func putBuffer(buf *golang_proto.Buffer) {
	_bufferPool.Put(buf)
}

func marshalJSON(message proto.Message, codec *codec) ([]byte, func(), error) {
	buf := bufferpool.Get()
	cleanup := func() { bufferpool.Put(buf) }
	if err := codec.jsonMarshaler.Marshal(buf, golang_proto.MessageV1(message)); err != nil {
		cleanup()
		return nil, nil, err
	}
	return buf.Bytes(), cleanup, nil
}
