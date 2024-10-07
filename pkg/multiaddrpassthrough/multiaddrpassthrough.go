// Copyright (c) 2024 Uber Technologies, Inc.
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

package multiaddrpassthrough

import (
	"errors"
	"strings"

	"google.golang.org/grpc/resolver"
)

func init() {
	resolver.Register(&multiaddrPassthroughBuilder{})
}

const Scheme = "multi-addr-passthrough"

var (
	errMissingAddr = errors.New("missing address")
)

type multiaddrPassthroughBuilder struct{}
type multiaddrPassthroughResolver struct{}

// NewBuilder creates a new multi address passthrough resolver builder.
func NewBuilder() resolver.Builder {
	return &multiaddrPassthroughBuilder{}
}

// Build creates and starts a multi address passthrough resolver.
// It expects the target to be a list of addresses on the format:
// multi-addr-passthrough:///192.168.0.1:2345/127.0.0.1:5678
func (*multiaddrPassthroughBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	addresses, err := parseTarget(target)
	if err != nil {
		return nil, err
	}

	err = cc.UpdateState(resolver.State{Addresses: addresses})
	if err != nil {
		return nil, err
	}

	return &multiaddrPassthroughResolver{}, nil
}

func (*multiaddrPassthroughBuilder) Scheme() string {
	return Scheme
}

// ResolveNow is a noop for the multi address passthrough resolver.
func (*multiaddrPassthroughResolver) ResolveNow(resolver.ResolveNowOptions) {}

// Close is a noop for the multi address passthrough resolver.
func (*multiaddrPassthroughResolver) Close() {}

func parseTarget(target resolver.Target) ([]resolver.Address, error) {
	endpoints := strings.Split(target.URL.Path, "/")
	addresses := make([]resolver.Address, 0, len(endpoints))

	for _, endpoint := range endpoints {
		if len(endpoint) > 0 {
			addresses = append(addresses, resolver.Address{Addr: endpoint})
		}
	}

	if len(addresses) == 0 {
		return nil, errMissingAddr
	}
	return addresses, nil
}
