// Copyright (c) 2018 Uber Technologies, Inc.
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

// Package yarpchttp implements a YARPC transport based on the HTTP/1.1 protocol.
// The HTTP transport provides support for Unary RPCs.
//
// Usage
//
// To serve your YARPC application over HTTP, create an inbound with a
// listening address or listener.
//
//  router := yarpcrouter.NewMapRouter("my-service")
// 	inbound := yarpchttp.Inbound{
//      Addr: ":8080",
//      Router: router,
//  }
//  inbound.Start(ctx) // and error handling
//  defer inbound.Stop(ctx) // and error handling
//
// To make requests to a YARPC application that supports HTTP, you will need a
// dialer, an outbound, and a client.
//
//  dialer := yarpchttp.Dialer{}
//  dialer.Start(ctx) // and error handling
//  defer dialer.Stop(ctx) // and error handling
//
//  url, err := url.Parse("http://127.0.0.1:8080")
//  // and error handling
//
//  outbound := yarpchttp.Outbound{
//      Dialer: dialer,
//      URL: err,
//  }
//
//  client := yarpcraw.New(&yarpc.Client{
//      Caller: "myservice",
//      Service: "theirservice",
//      Unary: outbound,
//  })
//
// To use a load balancer or peer chooser in general, introduce a peer list
// between the outbound and dialer.  The peer chooser will obtain addresses and
// punch them into the Host of the URL.
//
//  dialer := yarpchttp.Dialer{}
//  dialer.Start(ctx) // and error handling
//  defer dialer.Stop(ctx) // and error handling
//
//  list := roundrobin.New(dialer)
//
//  outbound := yarpchttp.Outbound{
//      Chooser: list,
//      URL: &url.URL{Host: "127.0.0.1:8080"},
//  }
//
// Wire Representation
//
// YARPC requests and responses are sent as plain HTTP requests and responses.
// YARPC metadata is sent inside reserved HTTP headers. Application headers
// for requests and responses are sent as HTTP headers with the header names
// prefixed with a pre-defined string. See Constants for more information on
// the names of these headers. The request and response bodies are sent as-is
// in the HTTP request or response body.
//
// See Also
//
// YARPC Properties: https://github.com/yarpc/yarpc/blob/master/properties.md
package yarpchttp
