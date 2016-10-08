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

//go:generate mockgen -destination=transporttest/inbound.go -package=transporttest go.uber.org/yarpc/transport Inbound

// Inbound is a transport that knows how to receive requests for procedure calls.
type Inbound interface {
	// Starts accepting new requests and dispatches them using the given
	// service configuration.
	//
	// The function MUST return immediately, although it SHOULD block until
	// the inbound is ready to start accepting new requests.
	//
	// Implementations can assume that this function is called at most once.
	Start(service ServiceDetail, deps Deps) error

	// Stops the inbound. No new requests will be processed.
	//
	// This MAY block while the server drains ongoing requests.
	Stop() error

	// TODO some way for the inbound to expose the host and port it's
	// listening on
}

// ServiceDetail specifies the service that an Inbound must serve.
type ServiceDetail struct {
	// Name of the service being served.
	Name string

	// Registry of procedures that this service offers.
	Registry Registry
}
