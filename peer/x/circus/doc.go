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

// Package circus provides a high performance load balancer.
//
// The primary premise of the circus peer list is that a production peer list
// will typically only have two classes of peer: those that have a certain
// number of pending requests, and those that have one more than that.
//
// A heap does unnecessary book-keeping by tracking the exact number of pending
// requests on each peer and maintaining a partially sorted order on that
// figure.
// The YARPC peer heap does additional book keeping so that it degenerates to
// round-robin instead of random load distribution when load is distributed
// evenly.
//
// Making this assumption, we can use a pair of circular doubly linked lists to
// model the highly and lightly loaded peer classes and move peers between them
// only when they begin and end requests.
//
// When the list of lightly loaded peers is empty, we can swap the lists.
// This would be more likely when aggregate load is increasing.
//
// This is a four ring circus.
// Each ring is a circular doubly linked list with a head node.
// One ring for unavailable peers, those waiting for an open connection to
// become available.
// The second and third ring are for peers with more or fewer concurrent
// requests.
// A fourth ring tracks unused nodes.
//
// Using this structure, we can approximate the behavior of a heap and limit
// every choice to a fixed number of integer swaps.
//
// The second premise is that the peer list will never need to contain more
// than 256 peers.
// Although it is possible to have more, beyond that point, it is typically
// okay to sample.
//
// Making this assumption we can use a fixed allocation of 256 nodes shared by
// the four circular doubly linked lists and their head nodes.
// Because of the size of the allocation, the internal references of the list
// can just be one byte wide.
//
// A third idea is to model the internal tables as columns.
// In general, this has the benefit of separating some concerns and potentially
// reducing CPU cache misses.
// For example, the linked lists do not need to contain their corresponding
// data type and we dodge a genericity bullet.
// We use the linked lists to track allocation and the topology of rings, and
// the coresponding indices in other arrays to track peers and subscribers.
//
// The circus avoids allocations in general by having a fixed width and
// limiting itself to 256 peers.
//
// The circus tends to choose peers in round-robin order but will tend to favor
// peers with fewer concurrent requests.
package circus
