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

package peer_test

import (
	"fmt"
	"testing"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/pendingheap"
	"go.uber.org/yarpc/peer/randpeer"
	"go.uber.org/yarpc/peer/roundrobin"
	"go.uber.org/yarpc/peer/tworandomchoices"
	"go.uber.org/yarpc/yarpctest"
)

func newHeap(trans peer.Transport) peer.ChooserList   { return pendingheap.New(trans) }
func newTRC(trans peer.Transport) peer.ChooserList    { return tworandomchoices.New(trans) }
func newRR(trans peer.Transport) peer.ChooserList     { return roundrobin.New(trans) }
func newRandom(trans peer.Transport) peer.ChooserList { return randpeer.New(trans) }

var listFuncs = []struct {
	name    string
	newList func(peer.Transport) peer.ChooserList
}{
	{"roundrobin", newRR},
	{"random", newRandom},
	{"tworandom", newTRC},
	{"heap", newHeap},
}

func BenchmarkChooseFinish(t *testing.B) {
	variances := []int{
		1,
		1024,
	}

	sizes := []int{
		1,
		2,
		4,
		8,
		16,
		32,
		64,
		128,
		256,
		512,
		1024,
	}

	for _, lf := range listFuncs {
		for _, size := range sizes {
			for _, variance := range variances {
				t.Run(fmt.Sprintf("%s-%dpeers-%dvariance", lf.name, size, variance), func(t *testing.B) {
					yarpctest.BenchmarkPeerListChooseFinish(t, size, variance, lf.newList)
				})
			}
		}
	}
}

func BenchmarkUpdate(t *testing.B) {
	for _, status := range []peer.ConnectionStatus{
		peer.Unavailable,
		peer.Available,
	} {
		for _, lf := range listFuncs {
			t.Run(fmt.Sprintf("%s-%s", lf.name, status), func(t *testing.B) {
				yarpctest.BenchmarkPeerListUpdate(t, status, lf.newList)
			})
		}
	}
}

func BenchmarkNotifyStateChange(t *testing.B) {
	for _, lf := range listFuncs {
		t.Run(lf.name, func(t *testing.B) {
			yarpctest.BenchmarkPeerListNotifyStatusChanged(t, lf.newList)
		})
	}
}
