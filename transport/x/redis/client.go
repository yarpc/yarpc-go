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

package redis

import (
	"time"

	"go.uber.org/yarpc/api/transport"
)

// Client is a subset of redis commands used to manage a queue
type Client interface {
	transport.Lifecycle

	// LPush adds item to the queue
	LPush(queue string, item []byte) error
	// This MUST return an error if the blocking call does not receive an item
	// BRPopLPush moves an item from the primary queue into a processing list.
	// within the timeout.
	BRPopLPush(from, to string, timeout time.Duration) ([]byte, error)
	// LRem removes one item from the queue key
	LRem(queue string, item []byte) error

	// Endpoint returns the enpoint configured for this client.
	Endpoint() string

	// ConnectionState returns the status of the connection(s).
	ConnectionState() string
}
