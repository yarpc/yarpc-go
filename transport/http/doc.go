// Copyright (c) 2026 Uber Technologies, Inc.
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

// Package http implements a YARPC transport based on the HTTP/1.1 protocol.
// The HTTP transport provides first class support for Unary RPCs and
// experimental support for Oneway RPCs.
//
// # Usage
//
// An HTTP Transport must be constructed to use this transport.
//
//	httpTransport := http.NewTransport()
//
// To serve your YARPC application over HTTP, pass an HTTP inbound in your
// yarpc.Config.
//
//	myInbound := httpTransport.NewInbound(":8080")
//	dispatcher := yarpc.NewDispatcher(yarpc.Config{
//		Name: "myservice",
//		Inbounds: yarpc.Inbounds{myInbound},
//	})
//
// To make requests to a YARPC application that supports HTTP, pass an HTTP
// outbound in your yarpc.Config.
//
//	myserviceOutbound := httpTransport.NewSingleOutbound("http://127.0.0.1:8080")
//	dispatcher := yarpc.NewDispatcher(yarpc.Config{
//		Name: "myclient",
//		Outbounds: yarpc.Outbounds{
//			"myservice": {Unary: myserviceOutbound},
//		},
//	})
//
// Note that stopping an HTTP transport does NOT immediately terminate ongoing
// requests. Connections will remain open until all clients have disconnected.
//
// By default, peers retained through the same Transport are shared across
// outbounds that resolve to the same address. To give a logical outbound its
// own peers, and therefore its own connections, isolated from any other
// outbound retaining the same address, build its chooser with an isolated
// Dialer:
//
//	myChooser := peer.NewSingle(
//		hostport.Identify("127.0.0.1:8080"),
//		httpTransport.NewDialer().WithConnectionIsolation(),
//	)
//	myserviceOutbound := httpTransport.NewOutbound(myChooser)
//
// Outbounds built by the transport itself, with NewSingleOutbound or from
// configuration, have no chooser for the caller to build. Construct the
// transport with IsolateConnectionsPerOutbound(true) to give each of those
// outbounds its own peers, as above.
//
// # HTTP/2 Connection Pool
//
// Each peer of an outbound that uses UseHTTP2 owns a pool of HTTP/2
// connections instead of a single one, because a connection carries at most
// SETTINGS_MAX_CONCURRENT_STREAMS requests at a time. The pool dials its first
// connection on the peer's first request, opens another when every connection
// reaches ScaleUpThreshold of MaxConcurrentStreams, and drains and closes
// connections again as load falls, between MinConnections and MaxConnections.
// It is tuned with the MaxConcurrentStreams, ScaleUpThreshold, ScaleDownGap,
// MinConnections, MaxConnections, ConnIdleTimeout, ScalingMonitorInterval and
// WithDynamicConnectionScaling options, with the clientConnectionPool section
// of TransportConfig and OutboundConfig, and at runtime with
// WithGlobalLiveConnectionPoolProvider. A pool is never shared between peers,
// so use an isolated Dialer (above) when two outbounds to one address must
// not share connections; an outbound with its own clientConnectionPool section
// or live provider is isolated automatically. These behave
// the same as the options of the same name in the gRPC transport.
//
// # Configuration
//
// An HTTP Transport may be configured using YARPC's configuration system. See
// TransportConfig, InboundConfig, and OutboundConfig for details on the
// different configuration parameters supported by this transport.
//
// # Wire Representation
//
// YARPC requests and responses are sent as plain HTTP requests and responses.
// YARPC metadata is sent inside reserved HTTP headers. Application headers
// for requests and responses are sent as HTTP headers with the header names
// prefixed with a pre-defined string. See Constants for more information on
// the names of these headers. The request and response bodies are sent as-is
// in the HTTP request or response body.
//
// # See Also
//
// YARPC Properties: https://github.com/yarpc/yarpc/blob/master/properties.md
package http
