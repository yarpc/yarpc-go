// Copyright (c) 2017 Uber Technologies, Inc.
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

package joinencodings

import "fmt"

// Join joins a list of encodings as a English (Chicago Style Manual)
// string for presentation in diagnostic messages.
func Join(encs []string) string {
	switch len(encs) {
	case 0:
		return "no encodings"
	case 1:
		return fmt.Sprintf("%q", encs[0])
	case 2:
		return fmt.Sprintf("%q or %q", encs[0], encs[1])
	default:
		i := 1
		inner := ""
		for ; i < len(encs)-1; i++ {
			inner = fmt.Sprintf("%s, %q", inner, encs[i])
		}
		// first, inner, inner, or last
		return fmt.Sprintf("%q%s, or %q", encs[0], inner, encs[i])
	}
}
