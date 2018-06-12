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
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/yarpc/yarpctest"
)

func TestPendingHeapConfig(t *testing.T) {
	minus1, zero, twenty := -1, 0, 20
	tests := []struct {
		name    string
		cfg     Configuration
		wantErr bool
	}{
		{
			name: "no configuration",
		},
		{
			name: "negative capacity",
			cfg: Configuration{
				Capacity: &minus1,
			},
			wantErr: true,
		},
		{
			name: "zero capacity",
			cfg: Configuration{
				Capacity: &zero,
			},
			wantErr: true,
		},
		{
			name: "valid capacity",
			cfg: Configuration{
				Capacity: &twenty,
			},
		},
	}

	s := Spec()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			build := s.BuildPeerList.(func(Configuration, peer.Transport, *yarpcconfig.Kit) (peer.ChooserList, error))
			pl, err := build(tt.cfg, yarpctest.NewFakeTransport(), nil)

			if tt.wantErr {
				require.Error(t, err, "must not construct a peer list")

			} else {
				require.NoError(t, err)
				pl.Update(peer.ListUpdates{Additions: []peer.Identifier{hostport.PeerIdentifier("foo-host:port")}})
			}
		})
	}
}
