// Copyright (c) 2018 Uber Technologies, Inc.
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

package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImports(t *testing.T) {
	t.Run("void add", func(t *testing.T) {
		assert.Empty(t, Imports{}.Add(""))
		assert.Empty(t, Imports{}.Add("."))
	})
	t.Run("add with import conflicts", func(t *testing.T) {
		imports := Imports{}
		imports.Add("json")
		imports.Add("encoding/json")
		imports.Add("encodingjson")

		expected := map[string]string{
			"json":          "json",
			"encoding/json": "encodingjson",
			"encodingjson":  "_encodingjson",
		}

		assert.Equal(t, len(expected), len(imports))
		assert.Equal(t, expected, map[string]string(imports))
	})
}
