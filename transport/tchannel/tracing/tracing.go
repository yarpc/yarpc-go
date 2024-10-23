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

package tracing

import (
	"github.com/opentracing/opentracing-go"
	"strings"
	"sync"
)

const (
	tchannelTracingKeyPrefix      = "$tracing$"
	tchannelTracingKeyMappingSize = 100
)

// Ensure HeadersCarrier implements both opentracing.TextMapReader and opentracing.TextMapWriter
var _ opentracing.TextMapReader = HeadersCarrier{}
var _ opentracing.TextMapWriter = HeadersCarrier{}

// HeadersCarrier is a dedicated carrier for TChannel.
// When writing the tracing headers into headers, the $tracing$ prefix is added to each tracing header key.
// When reading the tracing headers from headers, the $tracing$ prefix is removed from each tracing header key.
type HeadersCarrier map[string]string

// ForeachKey iterates over all tracing headers in the carrier, applying the provided
// handler function to each header after stripping the $tracing$ prefix from the keys.
func (c HeadersCarrier) ForeachKey(handler func(string, string) error) error {
	for k, v := range c {
		if !strings.HasPrefix(k, tchannelTracingKeyPrefix) {
			continue
		}
		noPrefixKey := tchannelTracingKeyDecoding.mapAndCache(k)
		if err := handler(noPrefixKey, v); err != nil {
			return err
		}
	}
	return nil
}

// Set adds a tracing header to the carrier, prefixing the key with $tracing$ before storing it.
func (c HeadersCarrier) Set(key, value string) {
	prefixedKey := tchannelTracingKeyEncoding.mapAndCache(key)
	c[prefixedKey] = value
}

// tchannelTracingKeysMapping is to optimize the efficiency of tracing header key manipulations.
// The implementation is forked from tchannel-go: https://github.com/uber/tchannel-go/blob/dev/tracing_keys.go#L36
type tchannelTracingKeysMapping struct {
	sync.RWMutex
	mapping map[string]string
	mapper  func(key string) string
}

var tchannelTracingKeyEncoding = &tchannelTracingKeysMapping{
	mapping: make(map[string]string),
	mapper: func(key string) string {
		return tchannelTracingKeyPrefix + key
	},
}

var tchannelTracingKeyDecoding = &tchannelTracingKeysMapping{
	mapping: make(map[string]string),
	mapper: func(key string) string {
		return key[len(tchannelTracingKeyPrefix):]
	},
}

func (m *tchannelTracingKeysMapping) mapAndCache(key string) string {
	m.RLock()
	v, ok := m.mapping[key]
	m.RUnlock()
	if ok {
		return v
	}
	m.Lock()
	defer m.Unlock()
	if v, ok := m.mapping[key]; ok {
		return v
	}
	mappedKey := m.mapper(key)
	if len(m.mapping) < tchannelTracingKeyMappingSize {
		m.mapping[key] = mappedKey
	}
	return mappedKey
}
