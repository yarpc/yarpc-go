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
)

// contextKey is an unexported type for context keys defined in this package.
// This prevents collisions with keys defined in other packages.
type contextKey struct{}

// meterInfoKey is the context key for meter information.
var meterInfoKey = contextKey{}

// MeterInfo contains metrics-related information that can be passed through context.
type MeterInfo struct {
	// Edge contains all the pre-initialized metrics for this RPC call
	Edge *edge
}

// WithMeterInfo adds meter information to the context.
func WithMeterInfo(ctx context.Context, info *MeterInfo) context.Context {
	return context.WithValue(ctx, meterInfoKey, info)
}

// GetMeterInfo retrieves meter information from the context.
// Returns nil if no meter info is present.
func GetMeterInfo(ctx context.Context) *MeterInfo {
	info, ok := ctx.Value(meterInfoKey).(*MeterInfo)
	if !ok {
		return nil
	}
	return info
}
