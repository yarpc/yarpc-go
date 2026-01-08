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

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"go.uber.org/yarpc/yarpcerrors"
)

// Codec is the interface for custom serialization formats.
type Codec interface {
	// Name returns the name of the codec (e.g., "proto", "json").
	Name() string
	// Marshal serializes v into bytes. The cleanup function, if non-nil,
	// should be called when the returned bytes are no longer needed.
	Marshal(v any) (data []byte, cleanup func(), err error)
	// Unmarshal deserializes data into v.
	Unmarshal(data []byte, v any) error
}

// Global codec registry - map lookup only, no locks on hot path.
var _codecs = make(map[string]Codec)

func init() {
	_codecs[string(Encoding)] = &protoCodec{}
	_codecs[string(JSONEncoding)] = &jsonCodec{}
}

// RegisterCodec registers a custom codec for use with YARPC.
//
// Custom codecs can override the built-in protobuf and JSON codecs by
// registering with the same name ("proto" or "json").
//
// NOTE: This function is NOT thread-safe. It must only be called during
// application initialization (e.g., in an init() function), before any
// RPC calls are made.
func RegisterCodec(codec Codec) {
	_codecs[codec.Name()] = codec
}

// GetCodecForEncoding returns the codec registered for an encoding.
// Returns nil if no codec is registered.
func GetCodecForEncoding(enc transport.Encoding) Codec {
	return _codecs[string(enc)]
}

// GetCodecNames returns the names of all registered codecs.
func GetCodecNames() []transport.Encoding {
	names := make([]transport.Encoding, 0, len(_codecs))
	for name := range _codecs {
		names = append(names, transport.Encoding(name))
	}
	return names
}

// getCodec returns the codec for the given encoding.
func getCodec(enc transport.Encoding) Codec {
	return _codecs[string(enc)]
}

// protoCodec implements Codec for protobuf encoding.
type protoCodec struct{}

func (c *protoCodec) Name() string {
	return string(Encoding)
}

func (c *protoCodec) Marshal(v any) ([]byte, func(), error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, func() {}, newNotProtoMessageError(v)
	}
	buf := getBuffer()
	if err := buf.Marshal(msg); err != nil {
		putBuffer(buf)
		return nil, func() {}, err
	}
	return buf.Bytes(), func() { putBuffer(buf) }, nil
}

func (c *protoCodec) Unmarshal(data []byte, v any) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return newNotProtoMessageError(v)
	}
	return proto.Unmarshal(data, msg)
}

// jsonCodec implements Codec for JSON encoding of protobuf messages.
type jsonCodec struct{}

func (c *jsonCodec) Name() string {
	return string(JSONEncoding)
}

func (c *jsonCodec) Marshal(v any) ([]byte, func(), error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, func() {}, newNotProtoMessageError(v)
	}
	buf := bufferpool.Get()
	if err := defaultJSONMarshaler.Marshal(buf, msg); err != nil {
		bufferpool.Put(buf)
		return nil, func() {}, err
	}
	return buf.Bytes(), func() { bufferpool.Put(buf) }, nil
}

func (c *jsonCodec) Unmarshal(data []byte, v any) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return newNotProtoMessageError(v)
	}
	return defaultJSONUnmarshaler.Unmarshal(bytes.NewReader(data), msg)
}

// Default JSON marshaler/unmarshaler (without AnyResolver)
var (
	defaultJSONMarshaler   = &jsonpb.Marshaler{}
	defaultJSONUnmarshaler = &jsonpb.Unmarshaler{AllowUnknownFields: true}
)

func newNotProtoMessageError(v any) error {
	return yarpcerrors.Newf(yarpcerrors.CodeInternal, "expected proto.Message, got %T", v)
}

func newUnrecognizedEncodingError(encoding transport.Encoding) error {
	return yarpcerrors.Newf(yarpcerrors.CodeInternal, "encoding.Expect should have handled encoding %q but did not", encoding)
}
