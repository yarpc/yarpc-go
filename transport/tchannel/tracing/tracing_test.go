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

package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHeadersCarrierSetAndForeachKey tests the Set and ForeachKey methods of HeadersCarrier.
func TestHeadersCarrierSetAndForeachKey(t *testing.T) {
	headers := make(map[string]string)
	carrier := HeadersCarrier(headers)

	// Set a header value
	carrier.Set("test-key", "test-value")

	// Ensure the header is set correctly with prefix
	prefixedKey := tchannelTracingKeyPrefix + "test-key"
	assert.Equal(t, "test-value", headers[prefixedKey])

	// Test ForeachKey method
	handlerCalled := false
	err := carrier.ForeachKey(func(key, value string) error {
		handlerCalled = true
		assert.Equal(t, "test-key", key)
		assert.Equal(t, "test-value", value)
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, handlerCalled)
}

// TestTChannelTracingKeysMapping tests the tchannelTracingKeysMapping.
func TestTChannelTracingKeysMapping(t *testing.T) {
	m := &tchannelTracingKeysMapping{
		mapping: make(map[string]string),
		mapper: func(key string) string {
			return tchannelTracingKeyPrefix + key
		},
	}

	key := "test-key"
	mappedKey := m.mapAndCache(key)

	// Ensure the mapped key has the correct prefix
	assert.Equal(t, tchannelTracingKeyPrefix+key, mappedKey)

	// Ensure the key is cached
	cachedValue := m.mapAndCache(key)
	assert.Equal(t, mappedKey, cachedValue)
}
