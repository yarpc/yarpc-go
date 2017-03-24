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

package decode

import (
	"fmt"
	"log"
	"sort"
)

func ExampleDecode() {
	type Item struct {
		Key   string `config:"name"`
		Value string
	}

	var item Item
	err := Decode(&item, map[string]interface{}{
		"name":  "foo",
		"value": "bar",
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(item.Key, item.Value)
	// Output: foo bar
}

type StringSet map[string]struct{}

func (ss *StringSet) Decode(decode Into) error {
	var items []string
	if err := decode(&items); err != nil {
		return err
	}

	*ss = make(map[string]struct{})
	for _, item := range items {
		(*ss)[item] = struct{}{}
	}
	return nil
}

func ExampleDecode_decoder() {
	var ss StringSet

	err := Decode(&ss, []interface{}{"foo", "bar", "foo", "baz"})
	if err != nil {
		log.Fatal(err)
	}

	var items []string
	for item := range ss {
		items = append(items, item)
	}
	sort.Strings(items)

	fmt.Println(items)
	// Output: [bar baz foo]
}
