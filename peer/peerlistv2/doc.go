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

// Package peerlist provides a utility for managing peer availability with a
// separate implementation of peer selection from just among available peers.
// The peer list implements the peer.ChooserList interface and accepts a
// peer.ListImplementation to provide the implementation-specific concern of,
// for example, a *roundrobin.List.
//
// The example is an implementation of peer.ChooserList using a random peer selection
// strategy, returned by newRandomListImplementation(), implementing
// peerlist.Implementation.
//
//   type List struct {
//   	*peerlist.List
//   }
//
//   func New(transport peer.Transport) *List {
//   	return &List{
//   		List: peerlist.New(
//   			"random",
//   			transport,
//   			newRandomListImplementation(),
//   		),
//   	}
//   }
//
package peerlist
