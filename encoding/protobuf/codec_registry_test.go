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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
)

func TestCodecRegistry(t *testing.T) {
	// Test codec registration and retrieval
	t.Run("codecRegistration", func(t *testing.T) {
		// Create a mock codec
		mockCodec := &MockCodec{name: "test-codec"}

		// Register codec

		RegisterCodec(mockCodec)

		// Test retrieval by codec name
		retrieved, ok := GetCodecForEncoding("test-codec")
		assert.True(t, ok, "Should find registered codec")
		assert.Equal(t, mockCodec, retrieved, "Should return registered codec")

		// Test fallback for unknown encoding
		unknown, ok := GetCodecForEncoding("unknown-encoding")
		assert.False(t, ok, "Should return false for unknown encoding")
		assert.Nil(t, unknown, "Should return nil for unknown encoding")
	})

	// Test public API
	t.Run("publicAPI", func(t *testing.T) {
		mockCodec := &MockCodec{name: "public-test"}

		// Register codec
		RegisterCodec(mockCodec)

		// Test public GetCodecForEncoding function
		retrieved, ok := GetCodecForEncoding("public-test")
		assert.True(t, ok, "Public API should find registered codec")
		assert.Equal(t, mockCodec, retrieved, "Public API should return registered codec")
	})

	// Test codec interface compliance
	t.Run("codecInterface", func(t *testing.T) {
		mockCodec := &MockCodec{name: "interface-test"}

		// Test Marshal
		data, cleanup, err := mockCodec.Marshal([]byte("test-data"))
		assert.NoError(t, err)
		assert.NotNil(t, data)
		assert.Equal(t, []byte("test-data"), data)
		if cleanup != nil {
			cleanup()
		}

		// Test Unmarshal
		var result []byte
		err = mockCodec.Unmarshal([]byte("unmarshal-test"), &result)
		assert.NoError(t, err)
		assert.Equal(t, []byte("unmarshal-test"), result)

		// Test Name
		assert.Equal(t, transport.Encoding("interface-test"), mockCodec.Name())
	})

	// Test built-in codecs are registered
	t.Run("builtinCodecs", func(t *testing.T) {
		names := GetCodecNames()

		hasProto := false
		hasJSON := false
		for _, name := range names {
			if name == Encoding {
				hasProto = true
			}
			if name == JSONEncoding {
				hasJSON = true
			}
		}

		assert.True(t, hasProto, "Proto encoding should be registered")
		assert.True(t, hasJSON, "JSON encoding should be registered")
	})

	// Test custom codec overrides built-in
	t.Run("customCodecOverride", func(t *testing.T) {
		// Save original
		originalProto, _ := GetCodecForEncoding(Encoding)

		// Register a custom proto codec
		customProto := &MockCodec{name: string(Encoding)}
		RegisterCodec(customProto)

		// GetCodecForEncoding should return the custom one
		retrieved, ok := GetCodecForEncoding(Encoding)
		assert.True(t, ok, "Should find custom codec")
		assert.Equal(t, customProto, retrieved, "Custom codec should override built-in")

		// Reset by registering back the original
		RegisterCodec(originalProto)
	})
}

func TestBuiltInCodecs(t *testing.T) {
	// Test proto codec error handling
	t.Run("proto_marshal_invalid_type", func(t *testing.T) {
		codec, ok := GetCodecForEncoding(Encoding)
		require.True(t, ok)
		require.NotNil(t, codec)

		_, _, err := codec.Marshal("not a proto message")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected proto.Message")
	})

	t.Run("proto_unmarshal_invalid_type", func(t *testing.T) {
		codec, ok := GetCodecForEncoding(Encoding)
		require.True(t, ok)
		require.NotNil(t, codec)

		var notProto string
		err := codec.Unmarshal([]byte("data"), &notProto)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected proto.Message")
	})

	// Test JSON codec error handling
	t.Run("json_codec_name", func(t *testing.T) {
		codec, ok := GetCodecForEncoding(JSONEncoding)
		require.True(t, ok)
		require.NotNil(t, codec)

		// Test codec name
		assert.Equal(t, JSONEncoding, codec.Name())
	})

	t.Run("json_marshal_invalid_type", func(t *testing.T) {
		codec, ok := GetCodecForEncoding(JSONEncoding)
		require.True(t, ok)
		require.NotNil(t, codec)

		_, _, err := codec.Marshal("not a proto message")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected proto.Message")
	})

	t.Run("json_unmarshal_invalid_type", func(t *testing.T) {
		codec, ok := GetCodecForEncoding(JSONEncoding)
		require.True(t, ok)
		require.NotNil(t, codec)

		var notProto string
		err := codec.Unmarshal([]byte(`{"value":"test"}`), &notProto)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected proto.Message")
	})
}

// MockCodec implements protobuf.Codec for testing
type MockCodec struct {
	name string
}

func (m *MockCodec) Marshal(v any) ([]byte, func(), error) {
	switch value := v.(type) {
	case []byte:
		return value, nil, nil
	default:
		return nil, nil, fmt.Errorf("expected []byte but got %T", v)
	}
}

func (m *MockCodec) Unmarshal(data []byte, v any) error {
	switch value := v.(type) {
	case *[]byte:
		*value = data
		return nil
	default:
		return fmt.Errorf("expected *[]byte but got %T", v)
	}
}

func (m *MockCodec) Name() transport.Encoding {
	return transport.Encoding(m.name)
}
