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

package yarpctest

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/hostport"
)

// BenchmarkPeerListChooseFinish assesses the speed of choose + finish for each
// of the YARPC peer choosers, with varying numbers of peers, and varying
// latencies.
// In order to remove exogenous behaviors from the simulation, each new request
// may be finished any number of turns in the future.
// One random request will be finished for every new peer chosen.
// By increasing the size of the concurrency window, we can increase the
// variance of pending requests on each individual peer.
//
// Size is the number of peers in the list.
// For this benchmark, the number of peers is fixed.
// All peers are connected.
// Variance is the number of concurrent pending requests.
// this number is also fixed for the benchmark.
// Variance comes into play because the request that finishes is chosen
// randomly from the window of previously chosen peers.
func BenchmarkPeerListChooseFinish(t *testing.B, size int, variance int, newList func(peer.Transport) peer.ChooserList) {
	t.ReportAllocs()

	trans := NewFakeTransport()
	list := newList(trans)

	// Build a static membership for the list.
	var ids []peer.Identifier
	for i := 0; i < size; i++ {
		ids = append(ids, hostport.Identify(strconv.Itoa(i)))
	}
	err := list.Update(peer.ListUpdates{
		Additions: ids,
	})
	require.NoError(t, err)

	require.NoError(t, trans.Start())
	require.NoError(t, list.Start())
	defer func() {
		require.NoError(t, trans.Stop())
		require.NoError(t, list.Stop())
	}()

	req := &transport.Request{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build a bank of concurrent requests.
	finishers := make([]func(error), variance)
	{
		i := 0
		for {
			_, onFinish, _ := list.Choose(ctx, req)
			if onFinish != nil {
				finishers[i] = onFinish
				i++
				if i >= variance {
					break
				}
			}
		}
	}

	t.ResetTimer()
	for n := 0; n < t.N; n++ {
		_, onFinish, _ := list.Choose(ctx, req)
		if onFinish == nil {
			continue
		}
		index := rand.Intn(variance)
		finishers[index](nil)
		finishers[index] = onFinish
	}
}

// BenchmarkPeerListUpdate measures the performance of peer list updates.
func BenchmarkPeerListUpdate(t *testing.B, init peer.ConnectionStatus, newList func(peer.Transport) peer.ChooserList) {
	trans := NewFakeTransport(InitialConnectionStatus(init))
	list := newList(trans)
	rng := rand.NewSource(1)

	var oldBits int64

	t.ResetTimer()
	for n := 0; n < t.N; n++ {
		newBits := rng.Int63()
		additions := idsForBits(newBits &^ oldBits)
		removals := idsForBits(oldBits &^ newBits)
		err := list.Update(peer.ListUpdates{
			Additions: additions,
			Removals:  removals,
		})
		if err != nil {
			panic(fmt.Sprintf("benchmark invalidated by update error: %v", err))
		}
		oldBits = newBits
	}
}

// BenchmarkPeerListNotifyStatusChanged measures the performance of a peer list
// in the presence of rapidly changing network conditions.
func BenchmarkPeerListNotifyStatusChanged(t *testing.B, newList func(peer.Transport) peer.ChooserList) {
	trans := NewFakeTransport()
	list := newList(trans)
	rng := rand.NewSource(1)

	// Add all 63 peers.
	err := list.Update(peer.ListUpdates{
		Additions: bitIds[:],
	})
	if err != nil {
		panic(fmt.Sprintf("benchmark invalidated by update error: %v", err))
	}

	t.ResetTimer()
	// Divide N by numIds. Add one to round up from zero.
	for n := 0; n < t.N/numIds+1; n++ {
		bits := rng.Int63()
		for i := uint(0); i < numIds; i++ {
			bit := (1 << i) & bits
			if bit != 0 {
				trans.SimulateConnect(bitIds[i])
			} else {
				trans.SimulateDisconnect(bitIds[i])
			}
		}
	}
}
