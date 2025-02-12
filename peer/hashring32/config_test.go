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

package hashring32

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/yarpc/yarpctest"
)

func TestPendingHeapConfig(t *testing.T) {
	s := Spec(nil, nil)
	duration := time.Second * 1

	c := Config{
		OffsetHeader:            "offsetHeader",
		ReplicaDelimiter:        "#",
		NumReplicas:             5,
		NumPeersEstimate:        1000,
		AlternateShardKeyHeader: "test-header",
		DefaultChooseTimeout:    &duration,
	}
	build := s.BuildPeerList.(func(c Config, t peer.Transport, k *yarpcconfig.Kit) (peer.ChooserList, error))
	pl, err := build(c, yarpctest.NewFakeTransport(), nil)
	assert.NoError(t, err, "must construct a peer list")
	pl.Update(peer.ListUpdates{Additions: []peer.Identifier{hostport.PeerIdentifier("127.0.0.1:8080")}})
}

func TestOffsetGeneratorValueConfig(t *testing.T) {
	s := Spec(nil, nil)
	duration := time.Second * 1

	c := Config{
		OffsetGeneratorValue:    4,
		ReplicaDelimiter:        "#",
		NumReplicas:             5,
		NumPeersEstimate:        1000,
		AlternateShardKeyHeader: "test-header",
		DefaultChooseTimeout:    &duration,
	}
	build := s.BuildPeerList.(func(c Config, t peer.Transport, k *yarpcconfig.Kit) (peer.ChooserList, error))
	pl, err := build(c, yarpctest.NewFakeTransport(), nil)
	assert.NoError(t, err, "must construct a peer list")
	pl.Update(peer.ListUpdates{Additions: []peer.Identifier{hostport.PeerIdentifier("127.0.0.1:8080")}})
}
