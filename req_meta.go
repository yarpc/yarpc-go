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

package yarpc

import "go.uber.org/yarpc/api/transport"

// CallReqMeta contains information about an outgoing YARPC request.
type CallReqMeta interface {
	Procedure(string) CallReqMeta
	Headers(Headers) CallReqMeta
	ShardKey(string) CallReqMeta
	RoutingKey(string) CallReqMeta
	RoutingDelegate(string) CallReqMeta

	GetProcedure() string
	GetHeaders() Headers
	GetShardKey() string
	GetRoutingKey() string
	GetRoutingDelegate() string
}

// ReqMeta contains information about an incoming YARPC request.
type ReqMeta interface {
	Caller() string
	Encoding() transport.Encoding
	Headers() Headers
	Procedure() string
	Service() string
}

// NewReqMeta constructs a CallReqMeta with the given Context.
func NewReqMeta() CallReqMeta {
	return &callReqMeta{}
}

type callReqMeta struct {
	procedure     string
	headers       Headers
	shardKey      string
	routeKey      string
	routeDelegate string
}

func (r *callReqMeta) Procedure(p string) CallReqMeta {
	r.procedure = p
	return r
}

func (r *callReqMeta) Headers(h Headers) CallReqMeta {
	r.headers = h
	return r
}

func (r *callReqMeta) ShardKey(sk string) CallReqMeta {
	r.shardKey = sk
	return r
}

func (r *callReqMeta) RoutingKey(rk string) CallReqMeta {
	r.routeKey = rk
	return r
}

func (r *callReqMeta) RoutingDelegate(rd string) CallReqMeta {
	r.routeDelegate = rd
	return r
}

func (r *callReqMeta) GetProcedure() string {
	return r.procedure
}

func (r *callReqMeta) GetHeaders() Headers {
	return r.headers
}

func (r *callReqMeta) GetShardKey() string {
	return r.shardKey
}

func (r *callReqMeta) GetRoutingKey() string {
	return r.routeKey
}

func (r *callReqMeta) GetRoutingDelegate() string {
	return r.routeDelegate
}
