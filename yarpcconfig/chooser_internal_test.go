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

package yarpcconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/peer"
)

type testIdentifier string

func (i testIdentifier) Identifier() string { return string(i) }

func identifyTest(id string) peer.Identifier { return testIdentifier(id) }

func TestIdentifyAll(t *testing.T) {
	tests := []struct {
		msg   string
		peers []string
		want  []string
	}{
		{
			msg:   "empty",
			peers: nil,
			want:  []string{},
		},
		{
			msg:   "distinct addresses",
			peers: []string{"127.0.0.1:8080", "127.0.0.1:8081"},
			want:  []string{"127.0.0.1:8080#1", "127.0.0.1:8081#1"},
		},
		{
			msg:   "repeated address gets one peer per occurrence",
			peers: []string{"127.0.0.1:8080", "127.0.0.1:8080", "127.0.0.1:8080"},
			want:  []string{"127.0.0.1:8080#1", "127.0.0.1:8080#2", "127.0.0.1:8080#3"},
		},
		{
			msg:   "interleaved duplicates keep list order",
			peers: []string{"a:1", "b:2", "a:1", "b:2", "a:1"},
			want:  []string{"a:1#1", "b:2#1", "a:1#2", "b:2#2", "a:1#3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			pids := identifyAll(identifyTest, tt.peers)

			got := make([]string, 0, len(pids))
			for _, pid := range pids {
				got = append(got, pid.Identifier())
			}
			assert.Equal(t, tt.want, got)

			// Identifiers must be unique: the peer list de-duplicates by
			// identifier, so a collision silently drops a peer.
			seen := make(map[string]struct{}, len(got))
			for _, id := range got {
				_, dup := seen[id]
				assert.False(t, dup, "duplicate identifier %q", id)
				seen[id] = struct{}{}
			}
		})
	}
}
