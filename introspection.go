// Copyright (c) 2017 Uber Technologies, Inc.
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

package yarpc

import (
	"fmt"

	"go.uber.org/yarpc/internal/introspection"
)

// Introspect returns detailed information about the dispatcher. This function
// acquires a lots of locks throughout and should only be called with some
// reserve. This method is public merely for use by the package yarpcmeta. The
// result of this function is internal to yarpc anyway.
func (d *Dispatcher) Introspect() introspection.DispatcherStatus {
	var inbounds []introspection.InboundStatus
	for _, i := range d.inbounds {
		var status introspection.InboundStatus
		if i, ok := i.(introspection.IntrospectableInbound); ok {
			status = i.Introspect()
		} else {
			status = introspection.InboundStatus{
				Transport: "Introspection not supported",
			}
		}
		inbounds = append(inbounds, status)
	}
	var outbounds []introspection.OutboundStatus
	for outboundKey, o := range d.outbounds {
		var status introspection.OutboundStatus
		if o.Unary != nil {
			if o, ok := o.Unary.(introspection.IntrospectableOutbound); ok {
				status = o.Introspect()
			} else {
				status.Transport = "Introspection not supported"
			}
			status.RPCType = "unary"
		}
		if o.Oneway != nil {
			if o, ok := o.Oneway.(introspection.IntrospectableOutbound); ok {
				status = o.Introspect()
			} else {
				status.Transport = "Introspection not supported"
			}
			status.RPCType = "oneway"
		}
		status.Service = o.ServiceName
		status.OutboundKey = outboundKey
		outbounds = append(outbounds, status)
	}
	procedures := introspection.IntrospectProcedures(d.table.Procedures())
	return introspection.DispatcherStatus{
		Name:       d.name,
		ID:         fmt.Sprintf("%p", d),
		Procedures: procedures,
		Inbounds:   inbounds,
		Outbounds:  outbounds,
	}
}
