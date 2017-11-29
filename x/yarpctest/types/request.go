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

package types

import "go.uber.org/yarpc/x/yarpctest/api"

// Service is a concrete type that represents the "service" for a request.
// It can be used in multiple interfaces.
type Service struct {
	Service string
}

// ApplyRequest implements api.RequestOption
func (n *Service) ApplyRequest(opts *api.RequestOpts) {
	opts.GiveRequest.Service = n.Service
}

// ApplyClientStreamRequest implements api.ClientStreamRequestOption
func (n *Service) ApplyClientStreamRequest(opts *api.ClientStreamRequestOpts) {
	opts.GiveRequest.Meta.Service = n.Service
}

// Procedure is a concrete type that represents the "procedure" for a request.
// It can be used in multiple interfaces.
type Procedure struct {
	Procedure string
}

// ApplyRequest implements api.RequestOption
func (n *Procedure) ApplyRequest(opts *api.RequestOpts) {
	opts.GiveRequest.Procedure = n.Procedure
}

// ApplyClientStreamRequest implements api.ClientStreamRequestOption
func (n *Procedure) ApplyClientStreamRequest(opts *api.ClientStreamRequestOpts) {
	opts.GiveRequest.Meta.Procedure = n.Procedure
}

// ShardKey is a concrete type that represents the "shard key" for a request.
// It can be used in multiple interfaces.
type ShardKey struct {
	ShardKey string
}

// ApplyRequest implements api.RequestOption
func (n *ShardKey) ApplyRequest(opts *api.RequestOpts) {
	opts.GiveRequest.ShardKey = n.ShardKey
}

// ApplyClientStreamRequest implements api.ClientStreamRequestOption
func (n *ShardKey) ApplyClientStreamRequest(opts *api.ClientStreamRequestOpts) {
	opts.GiveRequest.Meta.ShardKey = n.ShardKey
}
