// Copyright (c) 2020 Uber Technologies, Inc.
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
	"io"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// codec is a private helper struct used to hold custom marshling behavior.
type codec struct {
	protoMarshaler   proto.MarshalOptions
	protoUnmarshaler proto.UnmarshalOptions

	jsonMarshaler   protojson.MarshalOptions
	jsonUnmarshaler protojson.UnmarshalOptions
}

// Resolver matches the interface defined by
// proto.UnmarshalOptions#Resolver.
type Resolver interface {
	protoregistry.MessageTypeResolver
	protoregistry.ExtensionTypeResolver
}

func newCodec(resolver Resolver) *codec {
	return &codec{
		protoMarshaler:   proto.MarshalOptions{Deterministic: true},
		protoUnmarshaler: proto.UnmarshalOptions{Resolver: resolver, DiscardUnknown: false},

		jsonMarshaler:   protojson.MarshalOptions{Resolver: resolver},
		jsonUnmarshaler: protojson.UnmarshalOptions{Resolver: resolver, DiscardUnknown: false},
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
		return codec.protoUnmarshaler.Unmarshal(body, message)
	case JSONEncoding:
		return codec.jsonUnmarshaler.Unmarshal(body, message)
	default:
		return yarpcerrors.Newf(yarpcerrors.CodeInternal, "encoding.Expect should have handled encoding %q but did not", encoding)
	}
}

func marshal(encoding transport.Encoding, message proto.Message, codec *codec) ([]byte, error) {
	switch encoding {
	case Encoding:
		return codec.protoMarshaler.Marshal(message)
	case JSONEncoding:
		return codec.jsonMarshaler.Marshal(message)
	default:
		return nil, yarpcerrors.Newf(yarpcerrors.CodeInternal, "encoding.Expect should have handled encoding %q but did not", encoding)
	}
}
