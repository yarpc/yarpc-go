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

package roundrobin

import (
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/x/config"
)

// Spec returns a configuration specification for the round-robin peer list
// implementation, making it possible to select the least recently chosen peer
// with transports that use outbound peer list configuration (like HTTP).
//
//  cfg := config.New()
//  cfg.MustRegisterPeerList(roundrobin.Spec())
//
// This enables the round-robin peer list:
//
//  outbounds:
//    otherservice:
//      unary:
//        http:
//          url: https://host:port/rpc
//          round-robin:
//            peers:
//              - 127.0.0.1:8080
//              - 127.0.0.1:8081
func Spec() config.PeerListSpec {
	return config.PeerListSpec{
		Name: "round-robin",
		BuildPeerList: func(c struct{}, t peer.Transport, k *config.Kit) (peer.ChooserList, error) {
			return New(t), nil
		},
	}
}
