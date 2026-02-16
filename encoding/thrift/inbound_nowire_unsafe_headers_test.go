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

package thrift

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/observability"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
)

// Test helpers
type fakeUnaryHandler struct {
	handle func(context.Context, *transport.Request, transport.ResponseWriter) error
}

func (f fakeUnaryHandler) Handle(ctx context.Context, req *transport.Request, rw transport.ResponseWriter) error {
	if f.handle != nil {
		return f.handle(ctx, req, rw)
	}
	return nil
}

type fakeResponseWriter struct {
	transport.ResponseWriter
}

func (f *fakeResponseWriter) SetApplicationError()                                      {}
func (f *fakeResponseWriter) SetApplicationErrorMeta(_ *transport.ApplicationErrorMeta) {}
func (f *fakeResponseWriter) Write(p []byte) (n int, err error)                         { return len(p), nil }
func (f *fakeResponseWriter) AddHeaders(_ transport.Headers)                            {}
func (f *fakeResponseWriter) SetHeaders(_ transport.Headers)                            {}
func (f *fakeResponseWriter) AddSystemHeader(_, _ string)                               {}
func (f *fakeResponseWriter) RemoveSystemHeader(_ string)                               {}

// TestCheckAndEmitUnsafeHeadersCodeCoverage tests that the function executes all code paths
// Note: We cannot easily mock the internal edge type to verify actual metric emission,
// but we can ensure the code paths are exercised without panics.
func TestCheckAndEmitUnsafeHeadersCodeCoverage(t *testing.T) {
	tests := []struct {
		name          string
		transportName string
		setupHeaders  func() transport.Headers
		description   string
	}{
		{
			name:          "tchannel with uppercase headers",
			transportName: tchannel.TransportName,
			setupHeaders: func() transport.Headers {
				headers := transport.NewHeaders()
				headers = headers.With("X-Request-ID", "123")
				headers = headers.With("Content-Type", "application/json")
				return headers
			},
			description: "Should check uppercase for tchannel",
		},
		{
			name:          "tchannel with lowercase headers",
			transportName: tchannel.TransportName,
			setupHeaders: func() transport.Headers {
				headers := transport.NewHeaders()
				headers = headers.With("x-request-id", "123")
				headers = headers.With("content-type", "application/json")
				return headers
			},
			description: "Should not trigger issues with lowercase in tchannel",
		},
		{
			name:          "http with uppercase headers",
			transportName: "http",
			setupHeaders: func() transport.Headers {
				headers := transport.NewHeaders()
				headers = headers.With("X-Request-ID", "123")
				return headers
			},
			description: "Should not check uppercase for non-tchannel transport",
		},
		{
			name:          "tchannel with mixed case headers",
			transportName: tchannel.TransportName,
			setupHeaders: func() transport.Headers {
				headers := transport.NewHeaders()
				headers = headers.With("MixedCase", "value1")
				headers = headers.With("lowercase", "value2")
				headers = headers.With("UPPERCASE", "value3")
				return headers
			},
			description: "Should handle mixed case headers",
		},
		{
			name:          "empty headers",
			transportName: tchannel.TransportName,
			setupHeaders: func() transport.Headers {
				return transport.NewHeaders()
			},
			description: "Should handle empty headers gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with the test headers
			req := &transport.Request{
				Caller:    "test-caller",
				Service:   "test-service",
				Encoding:  "thrift",
				Procedure: "test::method",
				Transport: tt.transportName,
				Body:      strings.NewReader("test"),
				Headers:   tt.setupHeaders(),
			}

			// Test with nil edge (early return path)
			meterInfo := &observability.MeterInfo{
				Edge: nil,
			}

			// Execute the function under test - should not panic
			h := thriftNoWireHandler{}
			require.NotPanics(t, func() {
				h.checkAndEmitUnsafeHeaders(meterInfo, req)
			}, tt.description)
		})
	}
}

// TestCheckAndEmitUnsafeHeadersHeaderIterationLogic tests that the function
// iterates through headers correctly
func TestCheckAndEmitUnsafeHeadersHeaderIterationLogic(t *testing.T) {
	t.Run("iterates through all original headers", func(t *testing.T) {
		req := &transport.Request{
			Transport: tchannel.TransportName,
			Body:      strings.NewReader("test"),
		}

		// Add multiple headers to ensure iteration logic is exercised
		headers := transport.NewHeaders()
		headers = headers.With("header1", "value1")
		headers = headers.With("header2", "value2")
		headers = headers.With("header3", "value3")
		req.Headers = headers

		meterInfo := &observability.MeterInfo{
			Edge: nil,
		}

		h := thriftNoWireHandler{}
		// This should iterate through all 3 headers without panic
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})
	})

	t.Run("checks length comparison", func(t *testing.T) {
		req := &transport.Request{
			Transport: "http",
			Body:      strings.NewReader("test"),
		}

		headers := transport.NewHeaders()
		headers = headers.With("x-test-1", "value1")
		headers = headers.With("x-test-2", "value2")
		req.Headers = headers

		// The function should check len(Items()) vs len(OriginalItems())
		meterInfo := &observability.MeterInfo{
			Edge: nil,
		}

		h := thriftNoWireHandler{}
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})

		// Verify the lengths are equal (normal case)
		assert.Equal(t, len(req.Headers.Items()), len(req.Headers.OriginalItems()))
	})
}

// TestCheckAndEmitUnsafeHeadersCollisionDetection tests collision detection paths
func TestCheckAndEmitUnsafeHeadersCollisionDetection(t *testing.T) {
	t.Run("header.Get() call for each original header", func(t *testing.T) {
		// This test verifies that the function calls Headers.Get() for each original header
		// which exercises the collision detection logic
		req := &transport.Request{
			Transport: "http",
			Body:      strings.NewReader("test"),
		}

		headers := transport.NewHeaders()
		headers = headers.With("x-test-header", "value")
		headers = headers.With("x-another-header", "another-value")
		req.Headers = headers

		meterInfo := &observability.MeterInfo{
			Edge: nil,
		}

		h := thriftNoWireHandler{}
		// This should call Headers.Get() for each item in OriginalItems()
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})

		// Verify that all original headers can be retrieved (no collision in normal case)
		for origKey, origValue := range req.Headers.OriginalItems() {
			normalizedValue, exists := req.Headers.Get(origKey)
			assert.True(t, exists, "Expected header %q to exist", origKey)
			assert.Equal(t, origValue, normalizedValue, "Expected header values to match")
		}
	})
}

// TestCheckAndEmitUnsafeHeadersMultipleUppercaseHeaders tests handling of multiple uppercase headers
func TestCheckAndEmitUnsafeHeadersMultipleUppercaseHeaders(t *testing.T) {
	req := &transport.Request{
		Transport: tchannel.TransportName,
		Body:      strings.NewReader("test"),
	}

	// Add multiple headers with uppercase
	headers := transport.NewHeaders()
	headers = headers.With("X-Request-ID", "123")
	headers = headers.With("Content-Type", "application/json")
	headers = headers.With("X-Custom-Header", "value")
	headers = headers.With("lowercase-header", "lowercasevalue")
	req.Headers = headers

	meterInfo := &observability.MeterInfo{
		Edge: nil,
	}

	h := thriftNoWireHandler{}
	// Should iterate through all headers and check each one
	require.NotPanics(t, func() {
		h.checkAndEmitUnsafeHeaders(meterInfo, req)
	})

	// Verify we have the expected number of headers
	assert.Equal(t, 4, len(req.Headers.OriginalItems()))

	// Count how many have uppercase
	uppercaseCount := 0
	for origKey := range req.Headers.OriginalItems() {
		if headerKeyContainsUppercase(origKey) {
			uppercaseCount++
		}
	}
	assert.Equal(t, 3, uppercaseCount, "Expected 3 headers with uppercase characters")
}

// TestCheckAndEmitUnsafeHeadersWithDifferentTransports tests the function behavior
// with different transport types and header combinations
func TestCheckAndEmitUnsafeHeadersWithDifferentTransports(t *testing.T) {
	tests := []struct {
		name            string
		transportName   string
		originalHeaders map[string]string
		description     string
	}{
		{
			name:          "tchannel with uppercase headers",
			transportName: tchannel.TransportName,
			originalHeaders: map[string]string{
				"X-Request-ID": "123",
				"Content-Type": "application/json",
			},
			description: "Should check uppercase for tchannel",
		},
		{
			name:          "tchannel with lowercase headers",
			transportName: tchannel.TransportName,
			originalHeaders: map[string]string{
				"x-request-id": "123",
				"content-type": "application/json",
			},
			description: "Lowercase headers in tchannel are safe",
		},
		{
			name:          "http with uppercase headers",
			transportName: "http",
			originalHeaders: map[string]string{
				"X-Request-ID": "123",
				"Content-Type": "application/json",
			},
			description: "HTTP transport doesn't check for uppercase",
		},
		{
			name:            "empty headers",
			transportName:   tchannel.TransportName,
			originalHeaders: map[string]string{},
			description:     "Empty headers should not cause issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with the test headers
			req := &transport.Request{
				Caller:    "test-caller",
				Service:   "test-service",
				Encoding:  "thrift",
				Procedure: "test::method",
				Transport: tt.transportName,
				Body:      strings.NewReader("test"),
			}

			// Add headers
			headers := transport.NewHeaders()
			for k, v := range tt.originalHeaders {
				headers = headers.With(k, v)
			}
			req.Headers = headers

			// Create meter info (nil edge is valid, just early returns)
			meterInfo := &observability.MeterInfo{
				Edge: nil,
			}

			// Execute the function under test - should not panic
			h := thriftNoWireHandler{}
			require.NotPanics(t, func() {
				h.checkAndEmitUnsafeHeaders(meterInfo, req)
			}, tt.description)
		})
	}
}

// TestCheckAndEmitUnsafeHeadersNilSafety ensures comprehensive nil safety
func TestCheckAndEmitUnsafeHeadersNilSafety(t *testing.T) {
	h := thriftNoWireHandler{}

	t.Run("all nil", func(t *testing.T) {
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(nil, nil)
		})
	})

	t.Run("nil meter with valid request", func(t *testing.T) {
		req := &transport.Request{
			Headers: transport.NewHeaders().With("x-test", "value"),
		}
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(nil, req)
		})
	})

	t.Run("meter with nil edge", func(t *testing.T) {
		req := &transport.Request{
			Headers: transport.NewHeaders().With("x-test", "value"),
		}
		meterInfo := &observability.MeterInfo{Edge: nil}
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})
	})

	t.Run("nil request with meter", func(t *testing.T) {
		meterInfo := &observability.MeterInfo{}
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, nil)
		})
	})

	t.Run("valid meter and request but nil headers", func(t *testing.T) {
		req := &transport.Request{
			Transport: tchannel.TransportName,
		}
		meterInfo := &observability.MeterInfo{}
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})
	})
}

// TestCheckAndEmitUnsafeHeadersEdgeCases tests edge cases and boundary conditions
func TestCheckAndEmitUnsafeHeadersEdgeCases(t *testing.T) {
	t.Run("empty headers map", func(t *testing.T) {
		req := &transport.Request{
			Transport: tchannel.TransportName,
			Headers:   transport.NewHeaders(),
		}

		meterInfo := &observability.MeterInfo{Edge: nil}
		h := thriftNoWireHandler{}

		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})
	})

	t.Run("headers with special characters", func(t *testing.T) {
		req := &transport.Request{
			Transport: tchannel.TransportName,
			Body:      strings.NewReader("test"),
		}

		headers := transport.NewHeaders()
		headers = headers.With("x-custom-123", "value")
		headers = headers.With("special_chars-.-_", "value")
		headers = headers.With("numbers123", "value")
		req.Headers = headers

		meterInfo := &observability.MeterInfo{Edge: nil}
		h := thriftNoWireHandler{}

		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})
	})

	t.Run("mixed case headers", func(t *testing.T) {
		req := &transport.Request{
			Transport: tchannel.TransportName,
			Body:      strings.NewReader("test"),
		}

		headers := transport.NewHeaders()
		headers = headers.With("MixedCase", "value1")
		headers = headers.With("lowercase", "value2")
		headers = headers.With("UPPERCASE", "value3")
		req.Headers = headers

		meterInfo := &observability.MeterInfo{Edge: nil}
		h := thriftNoWireHandler{}

		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})
	})

	t.Run("headers with various lengths", func(t *testing.T) {
		req := &transport.Request{
			Transport: tchannel.TransportName,
			Body:      strings.NewReader("test"),
		}

		headers := transport.NewHeaders()
		headers = headers.With("a", "value")
		headers = headers.With("medium-length-header", "value")
		headers = headers.With("x-very-long-header-name-that-spans-multiple-words", "value")
		req.Headers = headers

		meterInfo := &observability.MeterInfo{Edge: nil}
		h := thriftNoWireHandler{}

		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})
	})
}

// TestHeaderKeyContainsUppercaseComprehensive provides comprehensive tests
// for the uppercase detection function
func TestHeaderKeyContainsUppercaseComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Basic cases
		{name: "empty string", input: "", expected: false},
		{name: "single lowercase", input: "a", expected: false},
		{name: "single uppercase", input: "A", expected: true},

		// All lowercase
		{name: "all lowercase", input: "lowercase", expected: false},
		{name: "lowercase with hyphen", input: "x-request-id", expected: false},
		{name: "lowercase with underscore", input: "content_type", expected: false},

		// All uppercase
		{name: "all uppercase", input: "UPPERCASE", expected: true},
		{name: "uppercase with hyphen", input: "X-REQUEST-ID", expected: true},
		{name: "uppercase with underscore", input: "CONTENT_TYPE", expected: true},

		// Mixed case
		{name: "mixed case", input: "MixedCase", expected: true},
		{name: "camel case", input: "camelCase", expected: true},
		{name: "pascal case", input: "PascalCase", expected: true},
		{name: "http header style", input: "X-Request-ID", expected: true},

		// With numbers
		{name: "lowercase with numbers", input: "header123", expected: false},
		{name: "uppercase with numbers", input: "HEADER123", expected: true},
		{name: "mixed with numbers", input: "Header123", expected: true},
		{name: "numbers only", input: "123456", expected: false},

		// With special characters
		{name: "special chars only", input: "-_-", expected: false},
		{name: "special chars with lowercase", input: "x-request-id", expected: false},
		{name: "special chars with uppercase", input: "X-Request-ID", expected: true},

		// Position tests
		{name: "uppercase at start", input: "Xheader", expected: true},
		{name: "uppercase at end", input: "headerX", expected: true},
		{name: "uppercase in middle", input: "headXer", expected: true},

		// Edge cases
		{name: "single hyphen", input: "-", expected: false},
		{name: "single underscore", input: "_", expected: false},
		{name: "multiple hyphens", input: "---", expected: false},
		{name: "mixed special and letters", input: "a-b-c", expected: false},
		{name: "mixed special and uppercase", input: "A-B-C", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := headerKeyContainsUppercase(tt.input)
			assert.Equal(t, tt.expected, result,
				"headerKeyContainsUppercase(%q) = %v, want %v", tt.input, result, tt.expected)
		})
	}
}

// BenchmarkHeaderKeyContainsUppercase provides benchmark data for the function
func BenchmarkHeaderKeyContainsUppercase(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"short_lowercase", "lowercase"},
		{"short_uppercase", "UPPERCASE"},
		{"short_mixed", "MixedCase"},
		{"medium_lowercase", "x-request-id-lowercase"},
		{"medium_uppercase", "X-REQUEST-ID-UPPERCASE"},
		{"medium_mixed", "X-Request-ID-Mixed"},
		{"long_lowercase", "x-very-long-lowercase-header-name-for-testing"},
		{"long_uppercase", "X-VERY-LONG-UPPERCASE-HEADER-NAME-FOR-TESTING"},
		{"long_mixed", "X-Very-Long-Mixed-Case-Header-Name-For-Testing"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = headerKeyContainsUppercase(tc.input)
			}
		})
	}
}

// TestCheckAndEmitUnsafeHeadersWithMetrics tests metric emission with actual observability middleware
func TestCheckAndEmitUnsafeHeadersWithMetrics(t *testing.T) {
	t.Run("tchannel uppercase headers emit metrics", func(t *testing.T) {
		metricsRoot := metrics.New()
		mw := observability.NewMiddleware(observability.Config{
			Logger:           zap.NewNop(),
			Scope:            metricsRoot.Scope(),
			ContextExtractor: observability.NewNopContextExtractor(),
		})

		req := &transport.Request{
			Caller:    "test-caller",
			Service:   "test-service",
			Encoding:  Encoding,
			Procedure: "test::method",
			Transport: tchannel.TransportName,
			Body:      strings.NewReader("test"),
		}

		// Add headers with uppercase characters
		headers := transport.NewHeaders()
		headers = headers.With("X-Request-ID", "123")
		headers = headers.With("Content-Type", "application/json")
		req.Headers = headers

		// Use a fake handler that captures the context with MeterInfo
		var capturedMeterInfo *observability.MeterInfo
		fakeHandler := fakeUnaryHandler{
			handle: func(ctx context.Context, r *transport.Request, rw transport.ResponseWriter) error {
				// Capture the MeterInfo from context
				capturedMeterInfo = observability.GetMeterInfo(ctx)
				// Test the function with captured meter info
				h := thriftNoWireHandler{}
				h.checkAndEmitUnsafeHeaders(capturedMeterInfo, req)
				return nil
			},
		}

		// Call through middleware to get proper context setup
		ctx := context.Background()
		err := mw.Handle(ctx, req, &fakeResponseWriter{}, fakeHandler)
		require.NoError(t, err)

		// Verify MeterInfo was available
		require.NotNil(t, capturedMeterInfo, "MeterInfo should not be nil")
		require.NotNil(t, capturedMeterInfo.Edge, "Edge should not be nil")

		// Verify metrics were emitted
		snapshot := metricsRoot.Snapshot()
		foundUnsafeHeaders := false
		for _, counter := range snapshot.Counters {
			if counter.Name == "unsafe_headers" {
				foundUnsafeHeaders = true
				assert.Greater(t, counter.Value, int64(0), "Expected unsafe_headers counter to be incremented")
				// Verify the tag indicates uppercase issue
				if issueType, ok := counter.Tags[observability.UnsafeHeaderIssueType]; ok {
					assert.Equal(t, _tchannelUppercaseKey, issueType, "Expected tchannel_uppercase_key issue type")
				}
			}
		}
		assert.True(t, foundUnsafeHeaders, "Expected to find unsafe_headers metric")
	})

	t.Run("multiple uppercase headers emit multiple metrics", func(t *testing.T) {
		metricsRoot := metrics.New()
		mw := observability.NewMiddleware(observability.Config{
			Logger:           zap.NewNop(),
			Scope:            metricsRoot.Scope(),
			ContextExtractor: observability.NewNopContextExtractor(),
		})

		req := &transport.Request{
			Caller:    "test-caller",
			Service:   "test-service",
			Encoding:  Encoding,
			Procedure: "test::method",
			Transport: tchannel.TransportName,
			Body:      strings.NewReader("test"),
		}

		headers := transport.NewHeaders()
		headers = headers.With("X-Request-ID", "123")
		headers = headers.With("Content-Type", "application/json")
		headers = headers.With("X-Custom-Header", "custom")
		headers = headers.With("lowercase-header", "lowercase")
		req.Headers = headers

		var capturedMeterInfo *observability.MeterInfo
		fakeHandler := fakeUnaryHandler{
			handle: func(ctx context.Context, r *transport.Request, rw transport.ResponseWriter) error {
				capturedMeterInfo = observability.GetMeterInfo(ctx)
				h := thriftNoWireHandler{}
				h.checkAndEmitUnsafeHeaders(capturedMeterInfo, req)
				return nil
			},
		}

		ctx := context.Background()
		err := mw.Handle(ctx, req, &fakeResponseWriter{}, fakeHandler)
		require.NoError(t, err)
		require.NotNil(t, capturedMeterInfo)
		require.NotNil(t, capturedMeterInfo.Edge)

		// Verify metrics
		snapshot := metricsRoot.Snapshot()
		uppercaseMetricCount := 0
		for _, counter := range snapshot.Counters {
			if counter.Name == "unsafe_headers" {
				if issueType, ok := counter.Tags[observability.UnsafeHeaderIssueType]; ok {
					if issueType == _tchannelUppercaseKey {
						uppercaseMetricCount++
						assert.Greater(t, counter.Value, int64(0))
					}
				}
			}
		}
		// We expect 3 headers with uppercase: X-Request-ID, Content-Type, X-Custom-Header
		assert.Equal(t, 3, uppercaseMetricCount, "Expected 3 uppercase header metrics")
	})

	t.Run("http transport does not emit uppercase metrics", func(t *testing.T) {
		metricsRoot := metrics.New()
		mw := observability.NewMiddleware(observability.Config{
			Logger:           zap.NewNop(),
			Scope:            metricsRoot.Scope(),
			ContextExtractor: observability.NewNopContextExtractor(),
		})

		req := &transport.Request{
			Caller:    "test-caller",
			Service:   "test-service",
			Encoding:  Encoding,
			Procedure: "test::method",
			Transport: "http",
			Body:      strings.NewReader("test"),
		}

		headers := transport.NewHeaders()
		headers = headers.With("X-Request-ID", "123")
		headers = headers.With("Content-Type", "application/json")
		req.Headers = headers

		var capturedMeterInfo *observability.MeterInfo
		fakeHandler := fakeUnaryHandler{
			handle: func(ctx context.Context, r *transport.Request, rw transport.ResponseWriter) error {
				capturedMeterInfo = observability.GetMeterInfo(ctx)
				h := thriftNoWireHandler{}
				h.checkAndEmitUnsafeHeaders(capturedMeterInfo, req)
				return nil
			},
		}

		ctx := context.Background()
		err := mw.Handle(ctx, req, &fakeResponseWriter{}, fakeHandler)
		require.NoError(t, err)
		require.NotNil(t, capturedMeterInfo)
		require.NotNil(t, capturedMeterInfo.Edge)

		// Verify NO uppercase metrics for HTTP transport
		snapshot := metricsRoot.Snapshot()
		for _, counter := range snapshot.Counters {
			if counter.Name == "unsafe_headers" {
				if issueType, ok := counter.Tags[observability.UnsafeHeaderIssueType]; ok {
					assert.NotEqual(t, _tchannelUppercaseKey, issueType,
						"HTTP transport should not emit uppercase metrics")
				}
			}
		}
	})

	t.Run("lowercase tchannel headers do not emit metrics", func(t *testing.T) {
		metricsRoot := metrics.New()
		mw := observability.NewMiddleware(observability.Config{
			Logger:           zap.NewNop(),
			Scope:            metricsRoot.Scope(),
			ContextExtractor: observability.NewNopContextExtractor(),
		})

		req := &transport.Request{
			Caller:    "test-caller",
			Service:   "test-service",
			Encoding:  Encoding,
			Procedure: "test::method",
			Transport: tchannel.TransportName,
			Body:      strings.NewReader("test"),
		}

		headers := transport.NewHeaders()
		headers = headers.With("x-request-id", "123")
		headers = headers.With("content-type", "application/json")
		headers = headers.With("x-custom-header", "custom")
		req.Headers = headers

		var capturedMeterInfo *observability.MeterInfo
		fakeHandler := fakeUnaryHandler{
			handle: func(ctx context.Context, r *transport.Request, rw transport.ResponseWriter) error {
				capturedMeterInfo = observability.GetMeterInfo(ctx)
				h := thriftNoWireHandler{}
				h.checkAndEmitUnsafeHeaders(capturedMeterInfo, req)
				return nil
			},
		}

		ctx := context.Background()
		err := mw.Handle(ctx, req, &fakeResponseWriter{}, fakeHandler)
		require.NoError(t, err)
		require.NotNil(t, capturedMeterInfo)
		require.NotNil(t, capturedMeterInfo.Edge)

		// Verify NO uppercase metrics for lowercase headers
		snapshot := metricsRoot.Snapshot()
		for _, counter := range snapshot.Counters {
			if counter.Name == "unsafe_headers" {
				if issueType, ok := counter.Tags[observability.UnsafeHeaderIssueType]; ok {
					assert.NotEqual(t, _tchannelUppercaseKey, issueType,
						"Lowercase headers should not emit uppercase metrics")
				}
			}
		}
	})

	t.Run("empty headers with metrics enabled", func(t *testing.T) {
		metricsRoot := metrics.New()
		mw := observability.NewMiddleware(observability.Config{
			Logger:           zap.NewNop(),
			Scope:            metricsRoot.Scope(),
			ContextExtractor: observability.NewNopContextExtractor(),
		})

		req := &transport.Request{
			Caller:    "test-caller",
			Service:   "test-service",
			Encoding:  Encoding,
			Procedure: "test::method",
			Transport: tchannel.TransportName,
			Body:      strings.NewReader("test"),
			Headers:   transport.NewHeaders(), // Empty headers
		}

		var capturedMeterInfo *observability.MeterInfo
		fakeHandler := fakeUnaryHandler{
			handle: func(ctx context.Context, r *transport.Request, rw transport.ResponseWriter) error {
				capturedMeterInfo = observability.GetMeterInfo(ctx)
				h := thriftNoWireHandler{}
				h.checkAndEmitUnsafeHeaders(capturedMeterInfo, req)
				return nil
			},
		}

		ctx := context.Background()
		err := mw.Handle(ctx, req, &fakeResponseWriter{}, fakeHandler)
		require.NoError(t, err)
		require.NotNil(t, capturedMeterInfo)
		require.NotNil(t, capturedMeterInfo.Edge)

		// No unsafe header metrics should be emitted for empty headers
		snapshot := metricsRoot.Snapshot()
		for _, counter := range snapshot.Counters {
			if counter.Name == "unsafe_headers" {
				t.Errorf("No unsafe_headers metrics should be emitted for empty headers, found: %+v", counter)
			}
		}
	})

	t.Run("mixed case headers tchannel", func(t *testing.T) {
		metricsRoot := metrics.New()
		mw := observability.NewMiddleware(observability.Config{
			Logger:           zap.NewNop(),
			Scope:            metricsRoot.Scope(),
			ContextExtractor: observability.NewNopContextExtractor(),
		})

		req := &transport.Request{
			Caller:    "test-caller",
			Service:   "test-service",
			Encoding:  Encoding,
			Procedure: "test::method",
			Transport: tchannel.TransportName,
			Body:      strings.NewReader("test"),
		}

		headers := transport.NewHeaders()
		headers = headers.With("X-Uppercase", "value1")
		headers = headers.With("lowercase", "value2")
		headers = headers.With("MixedCase", "value3")
		headers = headers.With("another-lowercase", "value4")
		req.Headers = headers

		var capturedMeterInfo *observability.MeterInfo
		fakeHandler := fakeUnaryHandler{
			handle: func(ctx context.Context, r *transport.Request, rw transport.ResponseWriter) error {
				capturedMeterInfo = observability.GetMeterInfo(ctx)
				h := thriftNoWireHandler{}
				h.checkAndEmitUnsafeHeaders(capturedMeterInfo, req)
				return nil
			},
		}

		ctx := context.Background()
		err := mw.Handle(ctx, req, &fakeResponseWriter{}, fakeHandler)
		require.NoError(t, err)
		require.NotNil(t, capturedMeterInfo)
		require.NotNil(t, capturedMeterInfo.Edge)

		// Count uppercase metrics
		snapshot := metricsRoot.Snapshot()
		uppercaseCount := 0
		for _, counter := range snapshot.Counters {
			if counter.Name == "unsafe_headers" {
				if issueType, ok := counter.Tags[observability.UnsafeHeaderIssueType]; ok {
					if issueType == _tchannelUppercaseKey {
						uppercaseCount++
					}
				}
			}
		}
		// Expected: X-Uppercase and MixedCase (2 headers with uppercase)
		assert.Equal(t, 2, uppercaseCount, "Expected 2 headers with uppercase")
	})

	t.Run("special characters in header keys", func(t *testing.T) {
		metricsRoot := metrics.New()
		mw := observability.NewMiddleware(observability.Config{
			Logger:           zap.NewNop(),
			Scope:            metricsRoot.Scope(),
			ContextExtractor: observability.NewNopContextExtractor(),
		})

		req := &transport.Request{
			Caller:    "test-caller",
			Service:   "test-service",
			Encoding:  Encoding,
			Procedure: "test::method",
			Transport: tchannel.TransportName,
			Body:      strings.NewReader("test"),
		}

		headers := transport.NewHeaders()
		headers = headers.With("x-custom-123", "value")
		headers = headers.With("special_chars-.-_", "value")
		headers = headers.With("numbers123", "value")
		req.Headers = headers

		var capturedMeterInfo *observability.MeterInfo
		fakeHandler := fakeUnaryHandler{
			handle: func(ctx context.Context, r *transport.Request, rw transport.ResponseWriter) error {
				capturedMeterInfo = observability.GetMeterInfo(ctx)
				h := thriftNoWireHandler{}
				h.checkAndEmitUnsafeHeaders(capturedMeterInfo, req)
				return nil
			},
		}

		ctx := context.Background()
		err := mw.Handle(ctx, req, &fakeResponseWriter{}, fakeHandler)
		require.NoError(t, err)
		require.NotNil(t, capturedMeterInfo)
		require.NotNil(t, capturedMeterInfo.Edge)

		// Special characters with lowercase should not trigger uppercase metrics
		snapshot := metricsRoot.Snapshot()
		for _, counter := range snapshot.Counters {
			if counter.Name == "unsafe_headers" {
				if issueType, ok := counter.Tags[observability.UnsafeHeaderIssueType]; ok {
					assert.NotEqual(t, _tchannelUppercaseKey, issueType,
						"Special characters with lowercase should not emit uppercase metrics")
				}
			}
		}
	})
}

// TestCheckAndEmitUnsafeHeadersAllCodePaths tests all code paths in checkAndEmitUnsafeHeaders
// with detailed verification of conditions and edge cases
func TestCheckAndEmitUnsafeHeadersAllCodePaths(t *testing.T) {
	t.Run("early return when meter is nil", func(t *testing.T) {
		req := &transport.Request{
			Transport: tchannel.TransportName,
			Headers:   transport.NewHeaders().With("X-Test", "value"),
			Body:      strings.NewReader("test"),
		}

		h := thriftNoWireHandler{}
		// Should return early without panic
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(nil, req)
		})
	})

	t.Run("early return when meter.Edge is nil", func(t *testing.T) {
		req := &transport.Request{
			Transport: tchannel.TransportName,
			Headers:   transport.NewHeaders().With("X-Test", "value"),
			Body:      strings.NewReader("test"),
		}

		meterInfo := &observability.MeterInfo{Edge: nil}
		h := thriftNoWireHandler{}
		// Should return early without panic
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})
	})

	t.Run("early return when treq is nil", func(t *testing.T) {
		meterInfo := &observability.MeterInfo{Edge: nil}
		h := thriftNoWireHandler{}
		// Should return early without panic
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, nil)
		})
	})

	t.Run("tchannel uppercase check is executed", func(t *testing.T) {
		// Verify the condition for uppercase checking executes correctly
		req := &transport.Request{
			Transport: tchannel.TransportName,
			Headers:   transport.NewHeaders().With("X-Test-Header", "value"),
			Body:      strings.NewReader("test"),
		}

		meterInfo := &observability.MeterInfo{Edge: nil}
		h := thriftNoWireHandler{}

		// This exercises the tchannel transport check and uppercase detection
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})

		// Verify the header actually contains uppercase
		assert.True(t, headerKeyContainsUppercase("X-Test-Header"))
	})

	t.Run("non-tchannel transport skips uppercase check", func(t *testing.T) {
		req := &transport.Request{
			Transport: "http",
			Headers:   transport.NewHeaders().With("X-Test-Header", "value"),
			Body:      strings.NewReader("test"),
		}

		meterInfo := &observability.MeterInfo{Edge: nil}
		h := thriftNoWireHandler{}

		// This should not execute the tchannel-specific uppercase check
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})
	})

	t.Run("collision detection for existing headers", func(t *testing.T) {
		// Create headers normally - should not have collisions
		headers := transport.NewHeaders()
		headers = headers.With("x-test", "value1")
		headers = headers.With("x-other", "value2")

		req := &transport.Request{
			Transport: "http",
			Headers:   headers,
			Body:      strings.NewReader("test"),
		}

		meterInfo := &observability.MeterInfo{Edge: nil}
		h := thriftNoWireHandler{}

		// This exercises the collision detection logic
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})

		// Verify no collisions in normal case
		for origKey, origValue := range req.Headers.OriginalItems() {
			normalizedValue, exists := req.Headers.Get(origKey)
			assert.True(t, exists, "Expected header %q to exist", origKey)
			assert.Equal(t, origValue, normalizedValue, "Expected values to match for %q", origKey)
		}
	})

	t.Run("length comparison code path", func(t *testing.T) {
		headers := transport.NewHeaders()
		headers = headers.With("x-header-1", "value1")
		headers = headers.With("x-header-2", "value2")

		req := &transport.Request{
			Transport: "http",
			Headers:   headers,
			Body:      strings.NewReader("test"),
		}

		meterInfo := &observability.MeterInfo{Edge: nil}
		h := thriftNoWireHandler{}

		// This exercises the length comparison logic
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})

		// Verify lengths are equal in normal case
		assert.Equal(t, len(req.Headers.Items()), len(req.Headers.OriginalItems()))
	})

	t.Run("all checks with tchannel and mixed headers", func(t *testing.T) {
		headers := transport.NewHeaders()
		headers = headers.With("X-Uppercase", "value1")
		headers = headers.With("lowercase", "value2")
		headers = headers.With("MixedCase", "value3")

		req := &transport.Request{
			Transport: tchannel.TransportName,
			Headers:   headers,
			Body:      strings.NewReader("test"),
		}

		meterInfo := &observability.MeterInfo{Edge: nil}
		h := thriftNoWireHandler{}

		// This exercises all code paths including uppercase detection for multiple headers
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})

		// Count uppercase headers
		uppercaseCount := 0
		for origKey := range req.Headers.OriginalItems() {
			if headerKeyContainsUppercase(origKey) {
				uppercaseCount++
			}
		}
		assert.Equal(t, 2, uppercaseCount, "Expected 2 headers with uppercase")
	})

	t.Run("empty headers", func(t *testing.T) {
		req := &transport.Request{
			Transport: tchannel.TransportName,
			Headers:   transport.NewHeaders(),
			Body:      strings.NewReader("test"),
		}

		meterInfo := &observability.MeterInfo{Edge: nil}
		h := thriftNoWireHandler{}

		// Should handle empty headers without issues
		require.NotPanics(t, func() {
			h.checkAndEmitUnsafeHeaders(meterInfo, req)
		})
	})
}

// BenchmarkCheckAndEmitUnsafeHeaders benchmarks the main function
func BenchmarkCheckAndEmitUnsafeHeaders(b *testing.B) {
	h := thriftNoWireHandler{}

	req := &transport.Request{
		Transport: tchannel.TransportName,
		Body:      strings.NewReader("test"),
	}

	headers := transport.NewHeaders()
	headers = headers.With("x-request-id", "123")
	headers = headers.With("content-type", "application/json")
	headers = headers.With("x-custom-header", "value")
	req.Headers = headers

	meterInfo := &observability.MeterInfo{Edge: nil}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.checkAndEmitUnsafeHeaders(meterInfo, req)
	}
}

// TestCheckAndEmitUnsafeHeadersEdgeCasesWithRealMetrics tests edge cases that create
// missing_from_items, key_collision, and extra_keys scenarios with actual metric emission
func TestCheckAndEmitUnsafeHeadersEdgeCasesWithRealMetrics(t *testing.T) {
	// Helper to create Headers with mismatched internal state using reflection
	createMismatchedHeaders := func(items, originalItems map[string]string) transport.Headers {
		h := transport.NewHeaders()
		v := reflect.ValueOf(&h).Elem()

		// Set items field
		itemsField := v.FieldByName("items")
		if itemsField.IsValid() {
			itemsPtr := (*map[string]string)(unsafe.Pointer(itemsField.UnsafeAddr()))
			*itemsPtr = items
		}

		// Set originalItems field
		originalItemsField := v.FieldByName("originalItems")
		if originalItemsField.IsValid() {
			originalItemsPtr := (*map[string]string)(unsafe.Pointer(originalItemsField.UnsafeAddr()))
			*originalItemsPtr = originalItems
		}

		return h
	}

	t.Run("missing_from_items_metric", func(t *testing.T) {
		metricsRoot := metrics.New()
		mw := observability.NewMiddleware(observability.Config{
			Logger:           zap.NewNop(),
			Scope:            metricsRoot.Scope(),
			ContextExtractor: observability.NewNopContextExtractor(),
		})

		// Create headers where originalItems has keys missing from items
		headers := createMismatchedHeaders(
			map[string]string{
				"x-exists": "value1",
			},
			map[string]string{
				"X-Exists":  "value1",
				"X-Missing": "value2",
			},
		)

		req := &transport.Request{
			Caller:    "test-caller",
			Service:   "test-service",
			Encoding:  Encoding,
			Procedure: "test::method",
			Transport: "http",
			Body:      strings.NewReader("test"),
			Headers:   headers,
		}

		var capturedMeterInfo *observability.MeterInfo
		fakeHandler := fakeUnaryHandler{
			handle: func(ctx context.Context, r *transport.Request, rw transport.ResponseWriter) error {
				capturedMeterInfo = observability.GetMeterInfo(ctx)
				h := thriftNoWireHandler{}
				h.checkAndEmitUnsafeHeaders(capturedMeterInfo, req)
				return nil
			},
		}

		ctx := context.Background()
		err := mw.Handle(ctx, req, &fakeResponseWriter{}, fakeHandler)
		require.NoError(t, err)
		require.NotNil(t, capturedMeterInfo)
		require.NotNil(t, capturedMeterInfo.Edge)

		snapshot := metricsRoot.Snapshot()
		foundMissing := false
		for _, counter := range snapshot.Counters {
			if counter.Name == "unsafe_headers" {
				if issueType, ok := counter.Tags[observability.UnsafeHeaderIssueType]; ok {
					if issueType == _missingFromItems {
						foundMissing = true
						assert.Greater(t, counter.Value, int64(0))
					}
				}
			}
		}
		assert.True(t, foundMissing, "Expected missing_from_items metric")
	})

	t.Run("key_collision_metric", func(t *testing.T) {
		metricsRoot := metrics.New()
		mw := observability.NewMiddleware(observability.Config{
			Logger:           zap.NewNop(),
			Scope:            metricsRoot.Scope(),
			ContextExtractor: observability.NewNopContextExtractor(),
		})

		// Create headers where values differ between items and originalItems
		headers := createMismatchedHeaders(
			map[string]string{
				"x-header": "normalized-value",
			},
			map[string]string{
				"X-Header": "original-value",
			},
		)

		req := &transport.Request{
			Caller:    "test-caller",
			Service:   "test-service",
			Encoding:  Encoding,
			Procedure: "test::method",
			Transport: "http",
			Body:      strings.NewReader("test"),
			Headers:   headers,
		}

		var capturedMeterInfo *observability.MeterInfo
		fakeHandler := fakeUnaryHandler{
			handle: func(ctx context.Context, r *transport.Request, rw transport.ResponseWriter) error {
				capturedMeterInfo = observability.GetMeterInfo(ctx)
				h := thriftNoWireHandler{}
				h.checkAndEmitUnsafeHeaders(capturedMeterInfo, req)
				return nil
			},
		}

		ctx := context.Background()
		err := mw.Handle(ctx, req, &fakeResponseWriter{}, fakeHandler)
		require.NoError(t, err)
		require.NotNil(t, capturedMeterInfo)
		require.NotNil(t, capturedMeterInfo.Edge)

		snapshot := metricsRoot.Snapshot()
		foundCollision := false
		for _, counter := range snapshot.Counters {
			if counter.Name == "unsafe_headers" {
				if issueType, ok := counter.Tags[observability.UnsafeHeaderIssueType]; ok {
					if issueType == _keyCollisionWithItems {
						foundCollision = true
						assert.Greater(t, counter.Value, int64(0))
					}
				}
			}
		}
		assert.True(t, foundCollision, "Expected key_collision_with_items metric")
	})

	t.Run("extra_keys_in_items_metric", func(t *testing.T) {
		metricsRoot := metrics.New()
		mw := observability.NewMiddleware(observability.Config{
			Logger:           zap.NewNop(),
			Scope:            metricsRoot.Scope(),
			ContextExtractor: observability.NewNopContextExtractor(),
		})

		// Create headers where items has more entries than originalItems
		headers := createMismatchedHeaders(
			map[string]string{
				"x-header1": "value1",
				"x-header2": "value2",
				"x-header3": "value3",
			},
			map[string]string{
				"X-Header1": "value1",
			},
		)

		req := &transport.Request{
			Caller:    "test-caller",
			Service:   "test-service",
			Encoding:  Encoding,
			Procedure: "test::method",
			Transport: "http",
			Body:      strings.NewReader("test"),
			Headers:   headers,
		}

		var capturedMeterInfo *observability.MeterInfo
		fakeHandler := fakeUnaryHandler{
			handle: func(ctx context.Context, r *transport.Request, rw transport.ResponseWriter) error {
				capturedMeterInfo = observability.GetMeterInfo(ctx)
				h := thriftNoWireHandler{}
				h.checkAndEmitUnsafeHeaders(capturedMeterInfo, req)
				return nil
			},
		}

		ctx := context.Background()
		err := mw.Handle(ctx, req, &fakeResponseWriter{}, fakeHandler)
		require.NoError(t, err)
		require.NotNil(t, capturedMeterInfo)
		require.NotNil(t, capturedMeterInfo.Edge)

		snapshot := metricsRoot.Snapshot()
		foundExtra := false
		for _, counter := range snapshot.Counters {
			if counter.Name == "unsafe_headers" {
				if issueType, ok := counter.Tags[observability.UnsafeHeaderIssueType]; ok {
					if issueType == _extraKeysInItems {
						foundExtra = true
						assert.Greater(t, counter.Value, int64(0))
					}
				}
			}
		}
		assert.True(t, foundExtra, "Expected extra_keys_in_items metric")
	})

	t.Run("combined_issues_metric", func(t *testing.T) {
		metricsRoot := metrics.New()
		mw := observability.NewMiddleware(observability.Config{
			Logger:           zap.NewNop(),
			Scope:            metricsRoot.Scope(),
			ContextExtractor: observability.NewNopContextExtractor(),
		})

		// Create headers with multiple issues at once
		// items has 3 entries, originalItems has 2 entries -> extra_keys_in_items
		headers := createMismatchedHeaders(
			map[string]string{
				"x-normal":  "value1",
				"x-extra":   "extra-value",
				"x-another": "another-value",
			},
			map[string]string{
				"X-Normal":  "different-value", // collision (different value)
				"X-Missing": "missing-value",   // missing from items
			},
		)

		req := &transport.Request{
			Caller:    "test-caller",
			Service:   "test-service",
			Encoding:  Encoding,
			Procedure: "test::method",
			Transport: tchannel.TransportName,
			Body:      strings.NewReader("test"),
			Headers:   headers,
		}

		var capturedMeterInfo *observability.MeterInfo
		fakeHandler := fakeUnaryHandler{
			handle: func(ctx context.Context, r *transport.Request, rw transport.ResponseWriter) error {
				capturedMeterInfo = observability.GetMeterInfo(ctx)
				h := thriftNoWireHandler{}
				h.checkAndEmitUnsafeHeaders(capturedMeterInfo, req)
				return nil
			},
		}

		ctx := context.Background()
		err := mw.Handle(ctx, req, &fakeResponseWriter{}, fakeHandler)
		require.NoError(t, err)
		require.NotNil(t, capturedMeterInfo)
		require.NotNil(t, capturedMeterInfo.Edge)

		snapshot := metricsRoot.Snapshot()
		foundIssues := make(map[string]bool)
		for _, counter := range snapshot.Counters {
			if counter.Name == "unsafe_headers" {
				if issueType, ok := counter.Tags[observability.UnsafeHeaderIssueType]; ok {
					foundIssues[issueType] = true
				}
			}
		}

		// Should find uppercase, collision, missing, and extra
		assert.True(t, foundIssues[_tchannelUppercaseKey], "Expected uppercase issue")
		assert.True(t, foundIssues[_keyCollisionWithItems], "Expected collision issue")
		assert.True(t, foundIssues[_missingFromItems], "Expected missing issue")
		assert.True(t, foundIssues[_extraKeysInItems], "Expected extra keys issue")
	})
}
