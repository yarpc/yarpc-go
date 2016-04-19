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

import "golang.org/x/net/context"

//go:generate mockgen -destination=transporttest/outbound.go -package=transporttest github.com/yarpc/yarpc-go/transport Outbound

// Outbound is a transport that knows how to send requests for procedure
// calls.
type Outbound interface {
	// Call sends the given request through this transport and returns its
	// response.
	Call(ctx context.Context, request *Request) (*Response, error)
}

// Outbounds is a map of service name to Outbound for that service.
type Outbounds map[string]Outbound

// Channel scopes an Outbound to a single caller-service pair.
type Channel struct {
	// Caller is the name of the service making the request.
	Caller string

	// Service is the name of the service to which the request is being made.
	Service string

	// Outbound is the transport used to send the request.
	Outbound Outbound

	// TODO: Can add caller-service-specific TTLs here. These can be inherited
	//from the YARPC otherwise.
}
