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

package noopresolver

import (
	"errors"

	"google.golang.org/grpc/resolver"
)

// Scheme is the scheme for the noop resolver.
const Scheme = "noop"

var errInvalidTarget = errors.New("noop resolver doesn't accept a target")

type noopBuilder struct{}

// NewBuilder creates a new noop resolver builder. This resolver won't resolve any address, so it expects the target to be empty.
// It is intended to be used by clients with custom resolution logic.
func NewBuilder() resolver.Builder {
	return &noopBuilder{}
}

func (*noopBuilder) Build(target resolver.Target, _ resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	if target.Endpoint() != "" || opts.Dialer != nil {
		return nil, errInvalidTarget
	}
	return &noopResolver{}, nil
}

func (*noopBuilder) Scheme() string {
	return Scheme
}

type noopResolver struct{}

func (*noopResolver) ResolveNow(_ resolver.ResolveNowOptions) {}

func (*noopResolver) Close() {}

func init() {
	resolver.Register(&noopBuilder{})
}
