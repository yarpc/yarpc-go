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

package yarpctchannel

import (
	"context"
	"net"

	opentracing "github.com/opentracing/opentracing-go"
	tchannel "github.com/uber/tchannel-go"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/zap"
)

// HeaderCase indicates the treatment of header case convention.
type HeaderCase int

const (
	// CanonicalHeaderCase indicates that YARPC will normalize the case of
	// headers to provide behavioral parity with other transport protocols like
	// HTTP.
	CanonicalHeaderCase HeaderCase = iota
	// OriginalHeaderCase indicates that YARPC will preserve the original
	// header case when forwarding headers for behavioral parity with a
	// TChannel proxy.
	OriginalHeaderCase
)

// Inbound receives YARPC requests over TChannel.
type Inbound struct {
	Service string

	// Addr specifies the port the TChannel should listen on.
	//
	// The default is ":0" (all interfaces, OS-assigned port).
	Addr string

	// Listener sets a net.Listener to use for the channel.
	//
	// If specified, Addr will be ignored.
	Listener net.Listener

	// Router receives inbound requests.
	Router yarpc.Router

	// HeaderCase specifies whether to forward headers without canonicalizing
	// their type case.
	HeaderCase HeaderCase

	// Tracer specifies the request tracer used for RPCs passing through the
	// TChannel inbound.
	Tracer opentracing.Tracer

	// Logger sets a logger to use for internal logging.
	//
	// The default is to not write any logs.
	Logger *zap.Logger

	ch *tchannel.Channel
}

// Start starts this Inbound.
func (i *Inbound) Start(_ context.Context) error {

	if i.Logger == nil {
		i.Logger = zap.NewNop()
	}

	if i.Tracer == nil {
		i.Tracer = opentracing.GlobalTracer()
	}

	chopts := tchannel.ChannelOptions{
		Tracer: i.Tracer,
		Handler: handler{
			router:     i.Router,
			tracer:     i.Tracer,
			logger:     i.Logger,
			headerCase: i.HeaderCase,
		},
	}
	ch, err := tchannel.NewChannel(i.Service, &chopts)
	if err != nil {
		return err
	}
	i.ch = ch

	if i.Listener == nil {
		if i.Addr == "" {
			addr, err := tchannel.ListenIP()
			if err != nil {
				return err
			}
			i.Addr = addr.String()
		}

		listener, err := net.Listen("tcp", i.Addr)
		if err != nil {
			return err
		}
		i.Listener = listener
	}

	err = i.ch.Serve(i.Listener)
	if err != nil {
		return err
	}

	i.Logger.Info("started TChannel inbound", zap.Stringer("address", i.Listener.Addr()))
	if i.Router == nil || len(i.Router.Procedures()) == 0 {
		i.Logger.Warn("no procedures specified for tchannel inbound")
	}

	return nil
}

// Stop stops the TChannel outbound.
func (i *Inbound) Stop(_ context.Context) error {
	return nil
}
