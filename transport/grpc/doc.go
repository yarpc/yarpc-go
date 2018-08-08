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

// Package grpc implements a YARPC transport based on the gRPC protocol.
// The gRPC transport provides support for Unary RPCs only.
//
// Usage
//
// A gRPC Transport must be constructed to use this transport.
//
//   grpcTransport := grpc.NewTransport()
//
// To serve your YARPC application over gRPC, pass a gRPC inbound in your
// yarpc.Config.
//
//   listener, err := net.Listen("tcp", ":8080")
//   if err != nil {
//     return err
//   }
//   myInbound := grpcTransport.NewInbound(listener)
//   dispatcher := yarpc.NewDispatcher(yarpc.Config{
//     Name: "myservice",
//     Inbounds: yarpc.Inbounds{myInbound},
//   })
//
// To configure TLS on your service listener, pass
// credentials.TransportCredentials as an InboundCredentials InboundOption.
// There are various ways to create credentials.TransportCredentials. See
// https://godoc.org/google.golang.org/grpc/credentials#TransportCredentials.
//
//   listener, err := net.Listen("tcp", ":4443")
//   if err != nil {
//     return err
//   }
//
//   myTLSConfig := &tls.Config{
//     // any arbitrary valid tls.Config
//   }
//   myTransportCredentials := credentials.NewTLS(myTLSConfig)
//   myInbound := grpcTransport.NewInbound(
//     listener,
//     InboundCredentials(myInboundCredentials),
//   )
//   dispatcher := yarpc.NewDispatcher(yarpc.Config{
//     Name: "myservice",
//     Inbounds: yarpc.Inbounds{myInbound},
//   })
//
// To make requests to a YARPC application that supports gRPC, pass a gRPC
// outbound in your yarpc.Config.
//
//   myserviceOutbound := grpcTransport.NewSingleOutbound("127.0.0.1:8080")
//   dispatcher := yarpc.NewDispatcher(yarpc.Config{
//     Name: "myclient",
//     Outbounds: yarpc.Outbounds{
//       "myservice": {Unary: myserviceOutbound},
//     },
//   })
//
// To make requests using TLS to an application supporting gRPC over TLS, pass
// credentials.TransportCredentials as a DialerCredentials DialOption. There
// are various ways to create credentials.TransportCredentials. See
// https://godoc.org/google.golang.org/grpc/credentials#TransportCredentials.
//
//   myTLSConfig := &tls.Config{
//     // any arbitrary valid tls.Config
//   }
//   myTransportCredentials := credentials.NewTLS(myTLSConfig)
//   myChooser := peer.NewSingle(
//     hostport.Identify("127.0.0.1:4443"),
//     grpcTransport.NewDialer(DialerCredentials(myTransportCredentials)),
//   )
//   myserviceOutbound := grpcTransport.NewOutbound(myChooser)
//   dispatcher := yarpc.NewDispatcher(yarpc.Config{
//     Name: "myclient",
//     Outbounds: yarpc.Outbounds{
//       "myservice": {Unary: myserviceOutbound},
//     },
//   })
//
// Configuration
//
// A gRPC transport may be configured using YARPC's configuration system.
// See TransportConfig, InboundConfig, and OutboundConfig for details on the
// different configuration parameters supported by this transport.
//
// See Also
//
// gRPC Project Page: https://grpc.io
// gRPC Wire Protocol Definition: https://grpc.io/docs/guides/wire.html
// gRPC Golang Library: https://github.com/grpc/grpc-go
package grpc
