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

package http

import (
	"net"
	"net/http"

	"github.com/yarpc/yarpc-go/transport"
)

// Inbound represents an HTTP Inbound. It is the same as the transport Inbound
// except it exposes the address on which the system is listening for
// connections.
type Inbound interface {
	transport.Inbound

	// Address on which the server is listening. Returns nil if Start has not
	// been called yet.
	Addr() net.Addr
}

// NewInbound builds a new HTTP inbound that listens on the given address.
func NewInbound(addr string) Inbound {
	return &inbound{addr: addr}
}

type inbound struct {
	addr     string
	listener net.Listener
}

func (i *inbound) Start(h transport.Handler) error {
	var err error
	i.listener, err = net.Listen("tcp", i.addr)
	if err != nil {
		return err
	}

	i.addr = i.listener.Addr().String() // in case it changed
	server := &http.Server{Handler: handler{h}}
	go server.Serve(i.listener)
	return nil
}

func (i *inbound) Stop() error {
	if i.listener == nil {
		return nil
	}
	err := i.listener.Close()
	i.listener = nil
	return err
}

func (i *inbound) Addr() net.Addr {
	if i.listener == nil {
		return nil
	}
	return i.listener.Addr()
}
