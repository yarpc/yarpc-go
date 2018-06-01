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

package pendingheap

import (
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/yarpc/yarpcerrors"
)

// Configuration descripes how to build a fewest pending heap peer list.
type Configuration struct {
	Capacity *int `config:"capacity"`
}

// Spec returns a configuration specification for the pending heap peer list
// implementation, making it possible to select the least recently chosen peer
// with transports that use outbound peer list configuration (like HTTP).
//
//  cfg := yarpcconfig.New()
//  cfg.MustRegisterPeerList(pendingheap.Spec())
//
// This enables the pending heap peer list:
//
//  outbounds:
//    otherservice:
//      unary:
//        http:
//          url: https://host:port/rpc
//          fewest-pending-requests:
//            capacity: 25
//            peers:
//              - 127.0.0.1:8080
//              - 127.0.0.1:8081
func Spec() yarpcconfig.PeerListSpec {
	return yarpcconfig.PeerListSpec{
		Name: "fewest-pending-requests",
		BuildPeerList: func(cfg Configuration, t peer.Transport, k *yarpcconfig.Kit) (peer.ChooserList, error) {
			if cfg.Capacity == nil {
				return New(t), nil
			}

			if *cfg.Capacity <= 0 {
				return nil, yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, "Capacity must be greater than 0")
			}

			return New(t, Capacity(*cfg.Capacity)), nil
		},
	}
}
