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

package transport

import "fmt"

// OutboundConfig is a configuration for how to call into another service.  It
// is used in conjunction with an encoding to send a request through one of the
// outbounds.
type OutboundConfig struct {
	CallerName  string
	Outbounds   Outbounds
}

// Caller is the name of the service making the request.
// implements ClientConfig#Caller (for backwards compatibility)
// TODO: This function should be deprecated, it's for legacy support.
// Use oc.CallerName instead
func (oc *OutboundConfig) Caller() string {
	return oc.CallerName
}

// Caller is the name of the service to which the request is being made.
// implements ClientConfig#Service (for backwards compatibility)
// TODO: This function should be deprecated, it's for legacy support.
// Use oc.Outbounds.ServiceName instead
func (oc *OutboundConfig) Service() string {
	return oc.Outbounds.ServiceName
}

// GetUnaryOutbound returns an outbound to send the request through or panics
// if there is no unary outbound for this service
// Implements ClientConfig#GetUnaryOutbound
// TODO: This function should be deprecated, it's for legacy support.
// Use oc.Outbounds.Unary instead (and panic if you want)
func (oc *OutboundConfig) GetUnaryOutbound() UnaryOutbound {
	if oc.Outbounds.Unary == nil {
		panic(fmt.Sprintf("Service %q does not have a unary outbound", oc.Outbounds.ServiceName))
	}
	return oc.Outbounds.Unary
}

// GetOnewayOutbound returns an outbound to send the request through or panics
// if there is no oneway outbound for this service
// Implements ClientConfig#GetOnewayOutbound
// TODO: This function should be deprecated, it's for legacy support.
// Use oc.Outbounds.Oneway instead (and panic if you want)
func (oc *OutboundConfig) GetOnewayOutbound() OnewayOutbound {
	if oc.Outbounds.Oneway == nil {
		panic(fmt.Sprintf("Service %q does not have a oneway outbound", oc.Outbounds.ServiceName))
	}

	return oc.Outbounds.Oneway
}
