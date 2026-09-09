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

package tracing

import (
	"strings"

	"go.opentelemetry.io/otel/propagation"
)

// Ensure OTelHeadersCarrier implements the OpenTelemetry carrier interface.
var _ propagation.TextMapCarrier = OTelHeadersCarrier{}

// OTelHeadersCarrier is the OpenTelemetry counterpart to HeadersCarrier.
//
// Like HeadersCarrier it adds the $tracing$ prefix to each key on write and
// strips it on read, so the propagator only ever sees undecorated header names.
// It shares the prefix mapping cache with HeadersCarrier.
type OTelHeadersCarrier map[string]string

// Get returns the value stored under the given unprefixed tracing key, or the
// empty string if the key is not present.
func (c OTelHeadersCarrier) Get(key string) string {
	return c[tchannelTracingKeyEncoding.mapAndCache(key)]
}

// Set stores value under the given key, adding the $tracing$ prefix.
func (c OTelHeadersCarrier) Set(key, value string) {
	c[tchannelTracingKeyEncoding.mapAndCache(key)] = value
}

// Keys returns the unprefixed names of every tracing header in the carrier.
// Non-tracing headers are skipped.
func (c OTelHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		if !strings.HasPrefix(k, tchannelTracingKeyPrefix) {
			continue
		}
		keys = append(keys, tchannelTracingKeyDecoding.mapAndCache(k))
	}
	return keys
}
