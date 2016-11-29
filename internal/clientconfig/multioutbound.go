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

package clientconfig

import (
	"fmt"

	"go.uber.org/yarpc/transport"
)

type multiOutbound struct {
	caller    string
	service   string
	Outbounds transport.Outbounds
}

// MultiOutbound constructs a ClientConfig backed by multiple outbound types
func MultiOutbound(caller, service string, Outbounds transport.Outbounds) transport.ClientConfig {
	return multiOutbound{caller: caller, service: service, Outbounds: Outbounds}
}

func (c multiOutbound) Caller() string  { return c.caller }
func (c multiOutbound) Service() string { return c.service }

func (c multiOutbound) GetUnaryOutbound() transport.UnaryOutbound {
	if c.Outbounds.Unary == nil {
		panic(fmt.Sprintf("Service %q does not have a unary outbound", c.service))
	}

	return c.Outbounds.Unary
}

func (c multiOutbound) GetOnewayOutbound() transport.OnewayOutbound {
	if c.Outbounds.Oneway == nil {
		panic(fmt.Sprintf("Service %q does not have a oneway outbound", c.service))
	}

	return c.Outbounds.Oneway
}
