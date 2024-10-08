// Copyright (c) 2024 Uber Technologies, Inc.
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
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
)

// TestGetPropagationFormat tests the correct format is returned based on the transport type.
func TestGetPropagationFormat(t *testing.T) {
	tests := []struct {
		name           string
		transport      string
		expectedFormat opentracing.BuiltinFormat
	}{
		{
			name:           "TChannel format",
			transport:      "tchannel",
			expectedFormat: opentracing.TextMap,
		},
		{
			name:           "HTTP format",
			transport:      "http",
			expectedFormat: opentracing.HTTPHeaders,
		},
		{
			name:           "gRPC format",
			transport:      "grpc",
			expectedFormat: opentracing.HTTPHeaders,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := GetPropagationFormat(tt.transport)
			assert.Equal(t, tt.expectedFormat, format)
		})
	}
}

// TestGetPropagationCarrier tests the correct carrier is returned based on the transport type.
func TestGetPropagationCarrier(t *testing.T) {
	headers := map[string]string{
		"key1": "value1",
	}

	tests := []struct {
		name         string
		transport    string
		expectedType interface{}
	}{
		{
			name:         "TChannel carrier",
			transport:    "tchannel",
			expectedType: TChannelHeadersCarrier(headers),
		},
		{
			name:         "HTTP carrier",
			transport:    "http",
			expectedType: opentracing.TextMapCarrier(headers),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			carrier := GetPropagationCarrier(headers, tt.transport)
			assert.IsType(t, tt.expectedType, carrier)
		})
	}
}

// TestTChannelHeadersCarrier tests the functionality of TChannelHeadersCarrier.
func TestTChannelHeadersCarrier(t *testing.T) {
	headers := map[string]string{
		"$tracing$trace-id": "1234",
		"$tracing$span-id":  "5678",
	}
	carrier := TChannelHeadersCarrier(headers)

	// Test ForeachKey
	t.Run("ForeachKey", func(t *testing.T) {
		keys := map[string]string{}
		err := carrier.ForeachKey(func(k, v string) error {
			keys[k] = v
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, "1234", keys["trace-id"])
		assert.Equal(t, "5678", keys["span-id"])
	})

	// Test Set
	t.Run("Set", func(t *testing.T) {
		carrier.Set("new-key", "new-value")
		assert.Equal(t, "new-value", carrier["$tracing$new-key"])
	})
}

// TestTChannelHeadersCarrierNoPrefix tests the carrier functionality without the tracing prefix.
func TestTChannelHeadersCarrierNoPrefix(t *testing.T) {
	headers := map[string]string{
		"no-prefix-key": "no-prefix-value",
	}
	carrier := TChannelHeadersCarrier(headers)

	// Test ForeachKey should not return headers without tracing prefix.
	t.Run("ForeachKey No Prefix", func(t *testing.T) {
		keys := map[string]string{}
		err := carrier.ForeachKey(func(k, v string) error {
			keys[k] = v
			return nil
		})

		assert.NoError(t, err)
		assert.Empty(t, keys)
	})

	// Test Set without prefix
	t.Run("Set Without Prefix", func(t *testing.T) {
		carrier.Set("new-key", "new-value")
		assert.Equal(t, "new-value", carrier["$tracing$new-key"])
	})
}

// TestTChannelTracingKeysMapping tests the key mapping optimizations.
func TestTChannelTracingKeysMapping(t *testing.T) {
	mapping := &tchannelTracingKeysMapping{
		mapping: make(map[string]string),
		mapper: func(key string) string {
			return tchannelTracingKeyPrefix + key
		},
	}

	// Test mapAndCache
	t.Run("MapAndCache", func(t *testing.T) {
		mapped := mapping.mapAndCache("test-key")
		assert.Equal(t, "$tracing$test-key", mapped)

		// Ensure it's cached
		cached := mapping.mapAndCache("test-key")
		assert.Equal(t, "$tracing$test-key", cached)
	})
}
