// Copyright (c) 2022 Uber Technologies, Inc.
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
	"fmt"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/yarpc/yarpcerrors"
)

// Configuration descripes how to build a fewest pending heap peer list.
type Configuration struct {
	Capacity *int `config:"capacity"`
	FailFast bool `config:"failFast"`
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
//            peers:
//              - 127.0.0.1:8080
//              - 127.0.0.1:8081
//
// Other than a specific peer or peers list, use any peer list updater
// registered with a yarpc Configurator.
// The configuration allows for alternative initial allocation capacity and a
// fail-fast option.
// With fail-fast enabled, the peer list will return an error immediately if no
// peers are available (connected) at the time the request is sent.
//
//  fewest-pending-requests:
//    peers:
//      - 127.0.0.1:8080
//    capacity: 1
//    failFast: true
func Spec() yarpcconfig.PeerListSpec {
	return SpecWithOptions()
}

// SpecWithOptions accepts additional list constructor options.
func SpecWithOptions(options ...ListOption) yarpcconfig.PeerListSpec {
	return yarpcconfig.PeerListSpec{
		Name: "fewest-pending-requests",
		BuildPeerList: func(cfg Configuration, t peer.Transport, k *yarpcconfig.Kit) (peer.ChooserList, error) {
			opts := make([]ListOption, 0, len(options)+2)

			opts = append(opts, options...)

			if cfg.Capacity != nil {
				if *cfg.Capacity <= 0 {
					return nil, yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument,
						fmt.Sprintf("Capacity must be greater than 0. Got: %d.", *cfg.Capacity))
				}
				opts = append(opts, Capacity(*cfg.Capacity))
			}

			if cfg.FailFast {
				opts = append(opts, FailFast())
			}

			return New(t, opts...), nil
		},
	}
}
