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

package yarpc

import (
	"fmt"
	"runtime"
	"sort"

	tchannel "github.com/uber/tchannel-go"
	thriftrw "go.uber.org/thriftrw/version"
	"go.uber.org/yarpc/internal/introspection"
	"google.golang.org/grpc"
)

// Introspect returns detailed information about the dispatcher. This function
// acquires a lots of locks throughout and should only be called with some
// reserve.
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
		if o.Unary != nil {
			var status introspection.OutboundStatus
			if o, ok := o.Unary.(introspection.IntrospectableOutbound); ok {
				status = o.Introspect()
			} else {
				status.Transport = "Introspection not supported"
			}
			status.RPCType = "unary"
			status.Service = o.ServiceName
			status.OutboundKey = outboundKey
			outbounds = append(outbounds, status)
		}
		if o.Oneway != nil {
			var status introspection.OutboundStatus
			if o, ok := o.Oneway.(introspection.IntrospectableOutbound); ok {
				status = o.Introspect()
			} else {
				status.Transport = "Introspection not supported"
			}
			status.RPCType = "oneway"
			status.Service = o.ServiceName
			status.OutboundKey = outboundKey
			outbounds = append(outbounds, status)
		}
	}

	sort.Sort(outboundStatuses(outbounds)) // keep debug pages deterministic

	procedures := introspection.IntrospectProcedures(d.table.Procedures())
	return introspection.DispatcherStatus{
		Name:            d.name,
		ID:              fmt.Sprintf("%p", d),
		Procedures:      procedures,
		Inbounds:        inbounds,
		Outbounds:       outbounds,
		PackageVersions: PackageVersions,
	}
}

// PackageVersions is a list of packages with corresponding versions.
var PackageVersions = []introspection.PackageVersion{
	{Name: "yarpc", Version: Version},
	{Name: "tchannel", Version: tchannel.VersionInfo},
	{Name: "thriftrw", Version: thriftrw.Version},
	{Name: "grpc-go", Version: grpc.Version},
	{Name: "go", Version: runtime.Version()},
}

type outboundStatuses []introspection.OutboundStatus

func (o outboundStatuses) Len() int {
	return len(o)
}
func (o outboundStatuses) Less(i, j int) bool {
	return o[i].OutboundKey < o[j].OutboundKey && o[i].RPCType < o[j].RPCType
}
func (o outboundStatuses) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}
