// Copyright (c) 2025 Uber Technologies, Inc.
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

package yarpctest

import (
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/x/yarpctest/types"
)

// Service specifies the "service" header for a request. It is a shared
// option across different requests.
func Service(service string) *types.Service {
	return &types.Service{Service: service}
}

// Procedure specifies the "procedure" header for a request. It is a shared
// option across different requests.
func Procedure(procedure string) *types.Procedure {
	return &types.Procedure{Procedure: procedure}
}

// ShardKey specifies that "shard key" header for a request. It is a shared
// option across different requests.
func ShardKey(key string) *types.ShardKey {
	return &types.ShardKey{ShardKey: key}
}

// Chooser overrides the peer.Chooser for a request.
func Chooser(f func(peer.Identifier, peer.Transport) (peer.Chooser, error)) *types.ChooserFactory {
	return &types.ChooserFactory{NewChooser: f}
}
