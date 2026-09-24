// Copyright (c) 2026 Uber Technologies, Inc.
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

package tworandomchoices

import (
	"fmt"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/yarpc/yarpcerrors"
)

// Configuration describes how to construct a two-random-choices peer list.
type Configuration struct {
	Capacity *int `config:"capacity"`
	FailFast bool `config:"failFast"`
}

// Spec returns a configuration specification for the "fewest pending requests
// of two random peers" implementation, making it possible to select the better
// of two random peer with transports that use outbound peer list configuration
// (like HTTP).
//
//	cfg := yarpcconfig.New()
//	cfg.MustRegisterPeerList(tworandomchoices.Spec())
//
// This enables the random peer list:
//
//	outbounds:
//	  otherservice:
//	    unary:
//	      http:
//	        url: https://host:port/rpc
//	        two-random-choices:
//	          peers:
//	            - 127.0.0.1:8080
//	            - 127.0.0.1:8081
func Spec() yarpcconfig.PeerListSpec {
	return SpecWithOptions()
}

// SpecWithOptions accepts additional list constructor options.
func SpecWithOptions(options ...ListOption) yarpcconfig.PeerListSpec {
	return yarpcconfig.PeerListSpec{
		Name: "two-random-choices",
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
