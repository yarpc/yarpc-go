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
