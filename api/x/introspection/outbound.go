// Copyright (c) 2025 Uber Technologies, Inc.
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

package introspection

// IntrospectableOutbound extends the Outbound interface.
type IntrospectableOutbound interface {
	Introspect() OutboundStatus
}

// OutboundStatus is a collection of basics info about an Outbound.
type OutboundStatus struct {
	Transport   string        `json:"transport"`
	RPCType     string        `json:"rpctype"`
	Endpoint    string        `json:"endpoint"`
	State       string        `json:"state"`
	Chooser     ChooserStatus `json:"chooser"`
	Service     string        `json:"service"`
	OutboundKey string        `json:"outboundkey"`
}

// OutboundStatusNotSupported is returned when not valid OutboundStatus can be
// produced.
var OutboundStatusNotSupported = OutboundStatus{}
