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

package yarpcerror

func validateName(name string) error {
	if name == "" {
		return nil
	}
	// I think that name is a global reference to a slice effectively, so len(name) has some overheade
	// https://stackoverflow.com/questions/26634554/go-multiple-len-calls-vs-performance
	// https://blog.golang.org/strings
	lenNameMinusOne := len(name) - 1
	for i, b := range name {
		if (i == 0 || i == lenNameMinusOne) && b == '-' {
			return Newf(CodeInternal, "invalid error name, must only start or end with lowercase letters: %s", name)
		}
		if !((b >= 'a' && b <= 'z') || b == '-') {
			return Newf(CodeInternal, "invalid error name, must only contain lowercase letters and dashes: %s", name)
		}
	}
	return nil
}
