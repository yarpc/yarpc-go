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

package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/net/metrics"
)

func TestReservedHeaderMetrics(t *testing.T) {
	m := metrics.New()

	t.Run("nil-scope", func(t *testing.T) {
		IncReservedHeaderStripped(nil, "", "")
		IncReservedHeaderError(nil, "", "")
	})

	t.Run("nil-counters", func(t *testing.T) {
		// Counters registration called only once
		registerHeaderMetrics(m.Scope())

		var (
			registeredStripped = reservedHeaderStripped
			registeredError    = reservedHeaderError
		)
		t.Cleanup(func() {
			reservedHeaderStripped = registeredStripped
			reservedHeaderError = registeredError
		})
		reservedHeaderStripped = nil
		reservedHeaderError = nil

		IncReservedHeaderStripped(m.Scope(), "", "")
		IncReservedHeaderError(m.Scope(), "", "")
	})

	t.Run("inc-header-metric", func(t *testing.T) {
		IncReservedHeaderStripped(m.Scope(), "source", "dest")
		IncReservedHeaderError(m.Scope(), "source", "dest")

		IncReservedHeaderStripped(m.Scope(), "source", "dest")
		IncReservedHeaderStripped(m.Scope(), "source", "dest-2")
		IncReservedHeaderStripped(m.Scope(), "source-2", "dest-2")

		s := m.Snapshot()

		var (
			strippedFound, errorFound bool
		)
		for _, c := range s.Counters {
			if c.Name == "reserved_headers_stripped" {
				strippedFound = true

				if c.Tags["source"] == "source" && c.Tags["dest"] == "dest" {
					assert.Equal(t, int64(2), c.Value)
				} else if c.Tags["source"] == "source" && c.Tags["dest"] == "dest-2" {
					assert.Equal(t, int64(1), c.Value)
				} else if c.Tags["source"] == "source-2" && c.Tags["dest"] == "dest-2" {
					assert.Equal(t, int64(1), c.Value)
				} else {
					t.Errorf("unexpected counter: %v", c)
				}
			} else if c.Name == "reserved_headers_error" {
				errorFound = true

				if c.Tags["source"] == "source" && c.Tags["dest"] == "dest" {
					assert.Equal(t, int64(1), c.Value)
				} else {
					t.Errorf("unexpected counter: %v", c)
				}
			} else {
				t.Errorf("unexpected counter: %v", c)
			}
		}

		assert.True(t, strippedFound)
		assert.True(t, errorFound)
	})
}
