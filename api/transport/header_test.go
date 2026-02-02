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

package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHeaders(t *testing.T) {
	tests := []struct {
		headers  map[string]string
		matches  map[string]string
		failures []string
	}{
		{
			nil,
			map[string]string{},
			[]string{"foo"},
		},
		{
			map[string]string{
				"Foo": "Bar",
				"Baz": "qux",
			},
			map[string]string{
				"foo": "Bar",
				"Foo": "Bar",
				"FOO": "Bar",
				"baz": "qux",
				"Baz": "qux",
				"BaZ": "qux",
			},
			[]string{"bar"},
		},
		{
			map[string]string{
				"foo": "bar",
				"baz": "",
			},
			map[string]string{
				"foo": "bar",
				"baz": "",
				"Baz": "",
			},
			[]string{"qux"},
		},
	}

	for _, tt := range tests {
		headers := HeadersFromMap(tt.headers)
		for k, v := range tt.matches {
			vg, ok := headers.Get(k)
			assert.True(t, ok, "expected true for %q", k)
			assert.Equal(t, v, vg, "value mismatch for %q", k)
		}
		for _, k := range tt.failures {
			v, ok := headers.Get(k)
			assert.False(t, ok, "expected false for %q", k)
			assert.Equal(t, "", v, "expected empty string for %q", k)
		}
	}
}

func TestItemsAndOriginalItems(t *testing.T) {
	type headers struct {
		key, val string
	}
	tests := []struct {
		msg                       string
		toDeleteKey               string
		headers                   []headers
		preDeletionItems          map[string]string
		postDeletionItems         map[string]string
		preDeletionOriginalItems  map[string]string
		postDeletionOriginalItems map[string]string
	}{
		{
			msg:         "delete lowercase/canonical key",
			toDeleteKey: "other-header",
			headers: []headers{
				{"foo-BAR-BaZ", "foo-bar-baz"},
				{"Foo-bAr-baZ", "FOO-BAR-BAZ"},
				{"other-header", "other-value"},
			},
			preDeletionItems: map[string]string{
				"foo-bar-baz":  "FOO-BAR-BAZ",
				"other-header": "other-value",
			},
			postDeletionItems: map[string]string{
				"foo-bar-baz": "FOO-BAR-BAZ",
			},
			preDeletionOriginalItems: map[string]string{
				"foo-BAR-BaZ":  "foo-bar-baz",
				"Foo-bAr-baZ":  "FOO-BAR-BAZ",
				"other-header": "other-value",
			},
			postDeletionOriginalItems: map[string]string{
				"foo-BAR-BaZ": "foo-bar-baz",
				"Foo-bAr-baZ": "FOO-BAR-BAZ",
			},
		},
		{
			msg:         "delete non-canonical key that does not exist in originalItem",
			toDeleteKey: "fOo-BAR-Baz",
			headers: []headers{
				{"foo-BAR-BaZ", "foo-bar-baz"},
				{"Foo-bAr-baZ", "FOO-BAR-BAZ"},
				{"other-header", "other-value"},
			},
			preDeletionItems: map[string]string{
				"foo-bar-baz":  "FOO-BAR-BAZ",
				"other-header": "other-value",
			},
			postDeletionItems: map[string]string{
				"other-header": "other-value",
			},
			preDeletionOriginalItems: map[string]string{
				"foo-BAR-BaZ":  "foo-bar-baz",
				"Foo-bAr-baZ":  "FOO-BAR-BAZ",
				"other-header": "other-value",
			},
			postDeletionOriginalItems: map[string]string{
				"foo-BAR-BaZ":  "foo-bar-baz",
				"Foo-bAr-baZ":  "FOO-BAR-BAZ",
				"other-header": "other-value",
			},
		},
		{
			msg:         "delete non-canonical key that also exists in originalItem",
			toDeleteKey: "foo-BAR-BaZ",
			headers: []headers{
				{"foo-BAR-BaZ", "foo-bar-baz"},
				{"Foo-bAr-baZ", "FOO-BAR-BAZ"},
				{"other-header", "other-value"},
			},
			preDeletionItems: map[string]string{
				"foo-bar-baz":  "FOO-BAR-BAZ",
				"other-header": "other-value",
			},
			postDeletionItems: map[string]string{
				"other-header": "other-value",
			},
			preDeletionOriginalItems: map[string]string{
				"foo-BAR-BaZ":  "foo-bar-baz",
				"Foo-bAr-baZ":  "FOO-BAR-BAZ",
				"other-header": "other-value",
			},
			postDeletionOriginalItems: map[string]string{
				"Foo-bAr-baZ":  "FOO-BAR-BAZ",
				"other-header": "other-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			header := NewHeaders()
			for _, v := range tt.headers {
				header = header.With(v.key, v.val)
			}

			assert.Equal(t, tt.preDeletionItems, header.Items())
			assert.Equal(t, tt.preDeletionOriginalItems, header.OriginalItems())

			header.Del(tt.toDeleteKey)
			assert.Equal(t, tt.postDeletionItems, header.Items())
			assert.Equal(t, tt.postDeletionOriginalItems, header.OriginalItems())
		})
	}
}

func TestEnableOverrideOriginalItemsWithCanonicalizedKeys(t *testing.T) {
	tests := []struct {
		msg                   string
		enableOverride        bool
		headers               []struct{ key, val string }
		expectedItems         map[string]string
		expectedOriginalItems map[string]string
	}{
		{
			msg:            "without override, original keys are preserved",
			enableOverride: false,
			headers: []struct{ key, val string }{
				{"Foo-Bar", "value1"},
				{"X-Custom-Header", "value2"},
			},
			expectedItems: map[string]string{
				"foo-bar":         "value1",
				"x-custom-header": "value2",
			},
			expectedOriginalItems: map[string]string{
				"Foo-Bar":         "value1",
				"X-Custom-Header": "value2",
			},
		},
		{
			msg:            "with override, original keys are canonicalized",
			enableOverride: true,
			headers: []struct{ key, val string }{
				{"Foo-Bar", "value1"},
				{"X-Custom-Header", "value2"},
			},
			expectedItems: map[string]string{
				"foo-bar":         "value1",
				"x-custom-header": "value2",
			},
			expectedOriginalItems: map[string]string{
				"foo-bar":         "value1",
				"x-custom-header": "value2",
			},
		},
		{
			msg:            "with override, duplicate keys with different casing are merged",
			enableOverride: true,
			headers: []struct{ key, val string }{
				{"Foo-Bar", "value1"},
				{"foo-bar", "value2"},
			},
			expectedItems: map[string]string{
				"foo-bar": "value2",
			},
			expectedOriginalItems: map[string]string{
				"foo-bar": "value2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			header := NewHeaders()
			if tt.enableOverride {
				header = header.EnableOverrideOriginalItemsWithCanonicalizedKeys()
			}
			for _, h := range tt.headers {
				header = header.With(h.key, h.val)
			}

			assert.Equal(t, tt.expectedItems, header.Items())
			assert.Equal(t, tt.expectedOriginalItems, header.OriginalItems())
		})
	}
}
