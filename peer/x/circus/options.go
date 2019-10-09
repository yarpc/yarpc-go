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

package circus

import "time"

// options accumulates the state arrived at by applying each of the option
// arguments of the list constructor.
type options struct {
	seed      int64
	noShuffle bool
	failFast  bool
}

var defaultOptions = options{
	seed: time.Now().UnixNano(),
}

// Option customizes the behavior of a peer circus.
type Option interface {
	apply(*options)
}

// optionFunc is a convenience for expressing options as closures without
// exposing the closure type in the public interface.
type optionFunc func(*options)

func (f optionFunc) apply(options *options) { f(options) }

// Seed specifies the seed for generating random choices.
func Seed(seed int64) Option {
	return optionFunc(func(options *options) {
		options.seed = seed
	})
}

// FailFast indicates that the peer list should not wait for a peer to become
// available when choosing a peer.
//
// This option is preferrable when the better failure mode is to retry from the
// origin, since another proxy instance might already have a connection.
func FailFast() Option {
	return optionFunc(func(options *options) {
		options.failFast = true
	})
}

// NoShuffle disables the default behavior of shuffling addition order.
func NoShuffle() Option {
	return optionFunc(func(options *options) {
		options.noShuffle = true
	})
}
