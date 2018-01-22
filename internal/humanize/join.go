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

package humanize

import "fmt"

// QuotedJoin transforms a list of terms into an English (Chicago Manual Style)
// serial comma delimited list with the given conjunction ("and", "or") and
// each term enquoted.
// It displays the "none" case if the list is empty, like "no soup".
func QuotedJoin(terms []string, conjunction, none string) string {
	switch len(terms) {
	case 0:
		return none
	case 1:
		return fmt.Sprintf("%q", terms[0])
	case 2:
		return fmt.Sprintf("%q %s %q", terms[0], conjunction, terms[1])
	default:
		i := 1
		inner := ""
		for ; i < len(terms)-1; i++ {
			inner = fmt.Sprintf("%s, %q", inner, terms[i])
		}
		// first, inner, inner, and/or last
		return fmt.Sprintf("%q%s, %s %q", terms[0], inner, conjunction, terms[i])
	}
}
