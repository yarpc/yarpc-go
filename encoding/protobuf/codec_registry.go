// Copyright (c) 2025 Uber Technologies, Inc.
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
	"sync"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/mem"
)

// Global codec registry for encoding-based codec overrides.
// Uses gRPC's encoding.CodecV2 interface for full codec v2 compatibility.
var (
	codecRegistryMutex sync.RWMutex
	codecRegistry      = make(map[string]encoding.CodecV2)
)

func init() {
	// Register built-in codecs at package initialization
	RegisterCodec(&protoCodec{})
	RegisterCodec(&jsonCodec{codec: newDefaultCodec()})
}

// RegisterCodec registers a codec for gRPC transport.
// Accepts any gRPC encoding.CodecV2 implementation.
func RegisterCodec(codec encoding.CodecV2) {
	codecRegistryMutex.Lock()
	defer codecRegistryMutex.Unlock()
	codecRegistry[codec.Name()] = codec
}

// getCodecForEncoding returns the registered codec for an encoding.
// For JSON encoding, uses the custom codec's marshalers if provided.
func getCodecForEncoding(encoding transport.Encoding, c *codec) encoding.CodecV2 {
	// If encoding is JSON and we have a custom codec, use it
	if encoding == JSONEncoding && c != nil {
		return &jsonCodec{codec: c}
	}

	codecRegistryMutex.RLock()
	defer codecRegistryMutex.RUnlock()

	return codecRegistry[string(encoding)]
}

// GetCodecForEncoding is the public version of getCodecForEncoding for testing/examples.
// It returns the codec registered in the registry without custom codec context.
func GetCodecForEncoding(encoding transport.Encoding) encoding.CodecV2 {
	return getCodecForEncoding(encoding, nil)
}

// GetCodecNames returns the names of all registered codecs
func GetCodecNames() []transport.Encoding {
	codecRegistryMutex.RLock()
	defer codecRegistryMutex.RUnlock()

	names := make([]transport.Encoding, 0, len(codecRegistry))
	for encoding := range codecRegistry {
		names = append(names, transport.Encoding(encoding))
	}
	return names
}

// protoCodec implements encoding.CodecV2 for protobuf encoding
type protoCodec struct{}

func (c *protoCodec) Marshal(v any) (mem.BufferSlice, error) {
	message, ok := v.(proto.Message)
	if !ok {
		return nil, proto.NewRequiredNotSetError("message is not a proto.Message")
	}

	data, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}

	return mem.BufferSlice{mem.SliceBuffer(data)}, nil
}

func (c *protoCodec) Unmarshal(data mem.BufferSlice, v any) error {
	message, ok := v.(proto.Message)
	if !ok {
		return proto.NewRequiredNotSetError("message is not a proto.Message")
	}
	return proto.Unmarshal(data.Materialize(), message)
}

func (c *protoCodec) Name() string {
	return string(Encoding)
}

// jsonCodec implements encoding.CodecV2 for JSON encoding
type jsonCodec struct {
	codec *codec
}

func (c *jsonCodec) Marshal(v any) (mem.BufferSlice, error) {
	message, ok := v.(proto.Message)
	if !ok {
		return nil, proto.NewRequiredNotSetError("message is not a proto.Message")
	}
	buf := bufferpool.Get()
	if err := c.codec.jsonMarshaler.Marshal(buf, message); err != nil {
		bufferpool.Put(buf)
		return nil, err
	}
	data := append([]byte(nil), buf.Bytes()...)
	bufferpool.Put(buf)
	return mem.BufferSlice{mem.SliceBuffer(data)}, nil
}

func (c *jsonCodec) Unmarshal(data mem.BufferSlice, v any) error {
	message, ok := v.(proto.Message)
	if !ok {
		return proto.NewRequiredNotSetError("message is not a proto.Message")
	}
	return c.codec.jsonUnmarshaler.Unmarshal(bytes.NewReader(data.Materialize()), message)
}

func (c *jsonCodec) Name() string {
	return string(JSONEncoding)
}

// codec is a private helper struct used to hold custom marshaling behavior for JSON.
type codec struct {
	jsonMarshaler   *jsonpb.Marshaler
	jsonUnmarshaler *jsonpb.Unmarshaler
}

// newDefaultCodec creates the default codec used for built-in JSON encoding.
func newDefaultCodec() *codec {
	return &codec{
		jsonMarshaler:   &jsonpb.Marshaler{},
		jsonUnmarshaler: &jsonpb.Unmarshaler{AllowUnknownFields: true},
	}
}

// newCodec creates a codec with a custom AnyResolver for JSON marshaling.
func newCodec(anyResolver jsonpb.AnyResolver) *codec {
	return &codec{
		jsonMarshaler:   &jsonpb.Marshaler{AnyResolver: anyResolver},
		jsonUnmarshaler: &jsonpb.Unmarshaler{AnyResolver: anyResolver, AllowUnknownFields: true},
	}
}
