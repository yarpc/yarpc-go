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
	"fmt"
	"strings"
)

// Imports maps a set of import paths to unique aliases.
type Imports map[string]string

// Add adds the path to the imports map, initially using the base directory
// as the package alias. If this alias is already in use, we continue to
// prepend the remaining filepath elements until we have receive a unique
// alias. If all of the path elements are exhausted, a '_' is continually used
// until we create a unique alias.
//
//   imports := NewImports("json")
//   imports.Add("encoding/json") -> "encodingjson"
//   imports.Add("encodingjson")  -> "_encodingjson"
func (imp Imports) Add(path string) string {
	if path == "" || path == "." {
		return ""
	}
	if alias, ok := imp[path]; ok {
		return alias
	}

	elems := strings.Split(path, "/")
	var alias string
	for i := 1; i <= len(elems); i++ {
		alias = strings.Join(elems[len(elems)-i:], "")
		if !imp.contains(alias) {
			break
		}
	}
	for imp.contains(alias) {
		alias = fmt.Sprintf("_%s", alias)
	}
	imp[path] = alias
	return alias
}

// contains determines whether the given alias is already
// registered in the import map.
func (imp Imports) contains(alias string) bool {
	for _, i := range imp {
		if i == alias {
			return true
		}
	}
	return false
}
