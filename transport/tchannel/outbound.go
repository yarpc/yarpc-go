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

package tchannel

import (
	"io"

	"github.com/yarpc/yarpc-go/transport"

	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
)

// OutboundOption configures Outbound.
type OutboundOption func(*outbound)

// HostPort specifies that the requests made by this outbound should be to
// the given address.
//
// By default, if HostPort was not specified, the Outbound will use the
// TChannel global peer list.
func HostPort(hostPort string) OutboundOption {
	return func(o *outbound) {
		o.HostPort = hostPort
	}
}

// NewOutbound builds a new TChannel outbound which uses the given Channel to
// make requests.
func NewOutbound(ch *tchannel.Channel, options ...OutboundOption) transport.Outbound {
	o := outbound{Channel: ch}
	for _, opt := range options {
		opt(&o)
	}
	return o
}

type outbound struct {
	Channel *tchannel.Channel

	// If specified, this is the address to which the request will be made.
	// Otherwise, the global peer list of the Channel will be used.
	HostPort string
}

func (o outbound) Call(ctx context.Context, req *transport.Request) (*transport.Response, error) {
	// NB(abg): Under the current API, the local service's name is required
	// twice: once when constructing the TChannel and then again when
	// constructing the RPC.

	var ch *tchannel.SubChannel
	if o.HostPort != "" {
		// We use an isolated subchannel when a HostPort is given so that this
		// hostport doesn't leak into the global peer list.
		ch = o.Channel.GetSubChannel(req.Service, tchannel.Isolated)
		// TODO(abg): Is this the best way to do this?
		ch.Peers().Add(o.HostPort)
	} else {
		ch = o.Channel.GetSubChannel(req.Service)
	}

	format := tchannel.Format(req.Encoding)

	call, err := ch.BeginCall(
		ctx, // TODO(abg): req.TTL what do
		req.Procedure,
		&tchannel.CallOptions{Format: format},
	)
	if err != nil {
		return nil, err
	}

	if err := writeHeaders(format, req.Headers, call.Arg2Writer); err != nil {
		return nil, err
	}

	if err := _writeBody(req.Body, call); err != nil {
		return nil, err
	}

	res := call.Response()

	// TODO(abg): handle errors

	headers, err := readHeaders(format, res.Arg2Reader)
	if err != nil {
		return nil, err
	}

	resBody, err := res.Arg3Reader()
	if err != nil {
		return nil, err
	}

	return &transport.Response{Headers: headers, Body: resBody}, nil
}

func _writeBody(body io.Reader, call *tchannel.OutboundCall) error {
	w, err := call.Arg3Writer()
	if err != nil {
		return err
	}

	if _, err := io.Copy(w, body); err != nil {
		return err
	}

	return w.Close()
}
