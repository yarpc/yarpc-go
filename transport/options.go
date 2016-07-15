// Copyright (c) 2016 Uber Technologies, Inc.
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

type optionsData map[interface{}]interface{}

// Options act as an extension point for transports to configure behavior of
// other parts of the system.
//
// A component that that wishes to be customizable based on transport.Options
// should declare a private type and key values off that.
//
// 	package foo
//
// 	type bar struct{}
//
// 	func OptionFoo(v string) (o transport.Options) {
// 		return o.With(bar{}, v)
// 	}
//
// A transport that wishes to change behavior simply needs to provide an
// Options object, merging OptionFoo into it.
//
// 	func (myOutbound) Options() (opts transport.Options) {
// 		return opts.Merge(foo.OptionFoo("hello"), bar.OptionBar(false))
// 	}
//
// Now the implementation of foo can use Options.Get to act differently based
// on the outbound's options.
type Options struct {
	data optionsData
}

// With returns a copy of this Options object with the given key-value pair
// added to it.
//
// The key should be a custom type to avoid conflicts with options of other
// components.
//
// 	opts = opts.With(foo{}, bar)
// 	opts = opts.With(baz{}, qux)
//
func (o Options) With(key, val interface{}) Options {
	data := make(optionsData, len(o.data)+1)
	for k, v := range o.data {
		data[k] = v
	}
	data[key] = val
	return Options{data}
}

// Get returns the value associated with the given key.
func (o Options) Get(k interface{}) (interface{}, bool) {
	if o.data == nil {
		return nil, false
	}
	v, ok := o.data[k]
	return v, ok
}

// Merge returns a copy of an Options object with items from all the given
// Options merged into it.
//
// Values in the rightmost Options object take precedence in case of
// conflicts.
func (o Options) Merge(others ...Options) Options {
	length := len(o.data)
	for _, other := range others {
		length += len(other.data)
	}

	data := make(optionsData, length)
	for k, v := range o.data {
		data[k] = v
	}
	for _, other := range others {
		for k, v := range other.data {
			data[k] = v
		}
	}

	return Options{data}
}
