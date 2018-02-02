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

// Package tchannel implements a YARPC transport based on the TChannel
// protocol. The TChannel transport provides support for Unary RPCs only.
//
// Usage
//
// A ChannelTransport must be constructed to use this transport. You can
// provide an existing TChannel Channel to construct the Channel transport.
//
// 	ch := getTChannelChannel()
// 	tchannelTransport, err := tchannel.NewChannelTransport(tchannel.WithChannel(ch))
//
// Alternatively, you can let YARPC own and manage the TChannel Channel for
// you by providing the service name. Note that this is the name of the local
// service, not the name of the service you will be sending requests to.
//
// 	tchannelTransport, err := tchannel.NewChannelTransport(tchannel.ServiceName("myservice"))
//
// To serve a YARPC application over TChannel, pass a TChannel inbound in your
// yarpc.Config.
//
// 	myInbound := tchannelTransport.NewInbound()
// 	dispatcher := yarpc.NewDispatcher(yarpc.Config{
// 		Name: "myservice",
// 		Inbounds: yarpc.Inbounds{myInbound},
// 	})
//
// To make requests to a YARPC application that supports TChannel, pass a
// TChannel outbound in your yarpc.Config.
//
// 	myserviceOutbound := tchannelTransport.NewOutbound()
// 	dispatcher := yarpc.NewDispatcher(yarpc.Config{
// 		Name: "myservice",
// 		Outbounds: yarpc.Outbounds{
// 			{Unary: myserviceOutbound},
// 		},
// 	})
//
// Configuration
//
// A TChannel transport may be configured using YARPC's configuration system.
// See TransportConfig, InboundConfig, and OutboundConfig for details on the
// different configuration parameters supported by this transport.
package tchannel
