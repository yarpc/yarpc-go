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

package direct

import (
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/yarpcconfig"
)

const name = "direct"

// Configuration describes how to build a direct peer chooser.
type Configuration struct{}

// Spec returns a configuration specification for the direct peer chooser. The
// chooser uses transport.Request#ShardKey as the peer dentifier.
//
//	cfg := yarpcconfig.New()
//	cfg.MustRegisterPeerChooser(direct.Spec())
//
// This enables the direct chooser:
//
//	outbounds:
//	  destination-service:
//	    grpc:
//	      direct: {}
func Spec(opts ...ChooserOption) yarpcconfig.PeerChooserSpec {
	return yarpcconfig.PeerChooserSpec{
		Name: name,
		BuildPeerChooser: func(cfg Configuration, t peer.Transport, _ *yarpcconfig.Kit) (peer.Chooser, error) {
			return New(cfg, t, opts...)
		},
	}
}
