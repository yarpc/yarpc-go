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

package transport

// Channel scopes outbounds to a single caller-service pair.
type Channel interface {
	// Name of the service making the request.
	Caller() string

	// Name of the service to which the request is being made.
	Service() string

	// Returns an outbound to send the request through.
	//
	// MAY be called multiple times for a request. MAY return different outbounds
	// for each call. The returned outbound MUST have already been started.
	GetOutbound() Outbound
}

// SimpleChannel constructs a Channel which always returns the same outbound.
func SimpleChannel(caller, service string, out Outbound) Channel {
	return simpleChannel{caller: caller, service: service, outbound: out}
}

type simpleChannel struct {
	caller   string
	service  string
	outbound Outbound
}

func (s simpleChannel) Caller() string        { return s.caller }
func (s simpleChannel) Service() string       { return s.service }
func (s simpleChannel) GetOutbound() Outbound { return s.outbound }
