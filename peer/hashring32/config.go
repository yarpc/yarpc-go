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

package hashring32

import (
	"go.uber.org/net/metrics"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hashring32/internal/farmhashring"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/zap"
)

// Config is the configuration object for hashring32yarpc
type Config struct {
	// OffsetHeader allows clients to pass in a header to adjust to offset value
	// in the Choose function.
	OffsetHeader string `config:"offsetHeader"`

	// PeerOverrideHeader allows clients to pass a header containing the shard
	// identifier for a specific peer to override the destination address for
	// the outgoing request.
	//
	// For example, if the peer list uses addresses to identify peers,
	// the hash ring will have retained a peer for every known address.
	// Specifying an address like "127.0.0.1" in the route override header will
	// deflect the request to that exact peer.
	// If that peer is not available, the request will continue on to the peer
	// implied by the shard key.
	PeerOverrideHeader string `config:"peerOverrideHeader"`

	ReplicaDelimiter string `config:"replicaDelimiter"`
}

// Spec returns a configuration specification for the hashed peer list
// implementation, making it possible to select peer based on a specified hashing
// function.
func Spec(logger *zap.Logger, meter *metrics.Scope) yarpcconfig.PeerListSpec {
	// TODO thread meter thorugh list options to abstract list metrics.

	return yarpcconfig.PeerListSpec{
		Name: "hashring32",
		BuildPeerList: func(c Config, t peer.Transport, k *yarpcconfig.Kit) (peer.ChooserList, error) {
			return New(
				t,
				farmhashring.Fingerprint32,
				OffsetHeader(c.OffsetHeader),
				ReplicaDelimiter(c.ReplicaDelimiter),
				PeerOverrideHeader(c.PeerOverrideHeader),
				Logger(logger),
			), nil
		},
	}
}
