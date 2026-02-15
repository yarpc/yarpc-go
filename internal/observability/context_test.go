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

package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMeterInfoContext(t *testing.T) {
	t.Run("WithMeterInfo and GetMeterInfo", func(t *testing.T) {
		ctx := context.Background()

		// Create mock edge (normally created by graph.begin)
		mockEdge := &edge{} // simplified for testing

		info := &MeterInfo{
			Edge: mockEdge,
		}

		// Store in context
		ctx = WithMeterInfo(ctx, info)

		// Retrieve from context
		retrieved := GetMeterInfo(ctx)

		assert.NotNil(t, retrieved)
		assert.Equal(t, info.Edge, retrieved.Edge)
	})

	t.Run("GetMeterInfo returns nil when not set", func(t *testing.T) {
		ctx := context.Background()

		retrieved := GetMeterInfo(ctx)

		assert.Nil(t, retrieved)
	})

	t.Run("Context isolation", func(t *testing.T) {
		ctx1 := context.Background()
		ctx2 := context.Background()

		info1 := &MeterInfo{
			Edge: &edge{},
		}

		ctx1 = WithMeterInfo(ctx1, info1)

		// ctx2 should not have meter info
		retrieved1 := GetMeterInfo(ctx1)
		retrieved2 := GetMeterInfo(ctx2)

		assert.NotNil(t, retrieved1)
		assert.Nil(t, retrieved2)
	})

	t.Run("Context chaining", func(t *testing.T) {
		ctx := context.Background()

		info := &MeterInfo{
			Edge: &edge{},
		}

		// Store meter info
		ctx = WithMeterInfo(ctx, info)

		type otherKey struct{}
		// Add other values to context
		ctx = context.WithValue(ctx, otherKey{}, "other_value")

		// Should still be able to retrieve meter info
		retrieved := GetMeterInfo(ctx)
		assert.NotNil(t, retrieved)
		assert.Equal(t, info.Edge, retrieved.Edge)

		// Other value should also be retrievable
		otherValue := ctx.Value(otherKey{})
		assert.Equal(t, "other_value", otherValue)
	})

	t.Run("Nil MeterInfo", func(t *testing.T) {
		ctx := context.Background()

		// Store nil meter info
		ctx = WithMeterInfo(ctx, nil)

		// Should return nil
		retrieved := GetMeterInfo(ctx)
		assert.Nil(t, retrieved)
	})

	t.Run("Overwrite MeterInfo", func(t *testing.T) {
		ctx := context.Background()

		edge1 := &edge{}
		edge2 := &edge{}

		info1 := &MeterInfo{
			Edge: edge1,
		}

		info2 := &MeterInfo{
			Edge: edge2,
		}

		// Store first meter info
		ctx = WithMeterInfo(ctx, info1)

		// Overwrite with second meter info
		ctx = WithMeterInfo(ctx, info2)

		// Should retrieve the second one
		retrieved := GetMeterInfo(ctx)
		assert.NotNil(t, retrieved)
		assert.Same(t, info2.Edge, retrieved.Edge, "Should retrieve the second edge")
		assert.NotSame(t, info1.Edge, retrieved.Edge, "Should not retrieve the first edge")
	})
}

func TestMeterInfoContextKey(t *testing.T) {
	t.Run("Context key type safety", func(t *testing.T) {
		ctx := context.Background()

		info := &MeterInfo{
			Edge: &edge{},
		}

		ctx = WithMeterInfo(ctx, info)

		// Try to retrieve with wrong key type - should not work
		wrongValue := ctx.Value("meterInfoKey")
		assert.Nil(t, wrongValue)

		// Using GetMeterInfo should work
		retrieved := GetMeterInfo(ctx)
		assert.NotNil(t, retrieved)
	})
}

func BenchmarkMeterInfoContext(b *testing.B) {
	info := &MeterInfo{
		Edge: &edge{},
	}

	b.Run("WithMeterInfo", func(b *testing.B) {
		ctx := context.Background()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = WithMeterInfo(ctx, info)
		}
	})

	b.Run("GetMeterInfo", func(b *testing.B) {
		ctx := context.Background()
		ctx = WithMeterInfo(ctx, info)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = GetMeterInfo(ctx)
		}
	})

	b.Run("WithMeterInfo and GetMeterInfo", func(b *testing.B) {
		ctx := context.Background()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ctx := WithMeterInfo(ctx, info)
			_ = GetMeterInfo(ctx)
		}
	})
}
