// Copyright (c) 2020 Uber Technologies, Inc.
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

package interpolate

import "fmt"

var (
	resolver      = mapResolver(map[string]string{"name": "Inigo Montoya"})
	emptyResolver = mapResolver(map[string]string{})
)

func Example() {
	s, err := Parse("My name is ${name}")
	if err != nil {
		panic(err)
	}

	out, err := s.Render(resolver)
	if err != nil {
		panic(err)
	}

	fmt.Println(out)
	// Output: My name is Inigo Montoya
}

func Example_default() {
	s, err := Parse("My name is ${name:what}")
	if err != nil {
		panic(err)
	}

	out, err := s.Render(emptyResolver)
	if err != nil {
		panic(err)
	}

	fmt.Println(out)
	// Output: My name is what
}
