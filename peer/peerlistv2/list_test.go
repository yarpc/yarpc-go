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

package peerlist

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
)

const (
	id1 = hostport.PeerIdentifier("1.2.3.4:1234")
	id2 = hostport.PeerIdentifier("4.3.2.1:4321")
	id3 = hostport.PeerIdentifier("1.1.1.1:1111")
)

func TestValues(t *testing.T) {
	vs := values(map[string]peer.Identifier{})
	assert.Equal(t, []peer.Identifier{}, vs)

	vs = values(map[string]peer.Identifier{"_": id1, "__": id2})
	assert.Equal(t, 2, len(vs))
	assert.Contains(t, vs, id1)
	assert.Contains(t, vs, id2)
}

func TestShuffle(t *testing.T) {
	for _, test := range []struct {
		msg  string
		seed int64
		in   []peer.Identifier
		want []peer.Identifier
	}{
		{
			"empty",
			0,
			[]peer.Identifier{},
			[]peer.Identifier{},
		},
		{
			"some",
			0,
			[]peer.Identifier{id1, id2, id3},
			[]peer.Identifier{id2, id3, id1},
		},
		{
			"different seed",
			7,
			[]peer.Identifier{id1, id2, id3},
			[]peer.Identifier{id2, id1, id3},
		},
	} {
		t.Run(test.msg, func(t *testing.T) {
			randSrc := rand.NewSource(test.seed)
			assert.Equal(t, test.want, shuffle(randSrc, test.in))
		})
	}
}
