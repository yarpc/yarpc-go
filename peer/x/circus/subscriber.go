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

import "go.uber.org/yarpc/api/peer"

var _ peer.Subscriber = (*subscriber)(nil)

// subscriber gets passed to the transport to retain or release a particular
// peer, and receives connection status change notifications.
// These it forwards to the list to adjust its tables accordingly.
// The circus reuses subscribers by constructing a bank of them once up front
// and tracking which ones are in use.
//
// The subscriber is a "thunk", meaning it provides only very thin functions
// that add an argument and jump to the corresponding list functions.
type subscriber struct {
	list          *List
	index         uint8
	boundOnFinish func(error)
}

// NotifyStatusChanged receives a status update from the transport and forwards
// the notification to the circus, with itself so the circus knows where to
// find the corresponding entries in its own tables.
func (s *subscriber) NotifyStatusChanged(pid peer.Identifier) {
	s.list.notifyStatusChanged(s, pid)
}

// onFinish becomes a closure tracked on boundOnFinish.
// This is the closure that Choose() returns so the caller can inform the list
// when it has finished a request.
func (s *subscriber) onFinish(error) {
	s.list.onFinish(s)
}
