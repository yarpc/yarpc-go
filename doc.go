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

// Package yarpc provides the YARPC service framework.
//
// With hundreds to thousands of services communicating with RPC, transport
// protocols (like HTTP and TChannel), encoding protocols (like JSON or
// Thrift), and peer choosers are the concepts that vary year over year.
// Separating these concerns allows services to change transports and wire
// protocols without changing call sites or request handlers, build proxies and
// wire protocol bridges, or experiment with load balancing strategies.
// YARPC is a toolkit for services and proxies.
//
// YARPC breaks RPC into interchangeable encodings, transports, and peer
// choosers.
// YARPC for Go provides reference implementations for HTTP/1.1, TChannel and gRPC
// transports, and also raw, JSON, Thrift, and Protobuf encodings.
// YARPC for Go provides a round robin peer chooser and experimental
// implementations for debug pages and rate limiting.
// YARPC for Go plans to provide a load balancer that uses a
// least-pending-requests strategy.
// Peer choosers can implement any strategy, including load balancing or sharding,
// in turn bound to any peer list updater.
//
// Regardless of transport, every RPC has some common properties: caller name,
// service name, procedure name, encoding name, deadline or TTL, headers,
// baggage (multi-hop headers), and tracing.
// Each RPC can also have an optional shard key, routing key, or routing
// delegate for advanced routing.
// YARPC transports use a shared API for capturing RPC metadata, so middleware
// can apply to requests over any transport.
//
// Each YARPC transport protocol can implement inbound handlers and outbound
// callers. Each of these can support different RPC types, like unary (request and
// response) or oneway (request and receipt) RPC. A future release of YARPC will
// add support for other RPC types including variations on streaming and pubsub.
package yarpc
