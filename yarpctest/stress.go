// Copyright (c) 2019 Uber Technologies, Inc.
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
	"math/rand"
	"strconv"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/peer/hostport"
)

// ListStressTest describes the parameters of a stress test for a peer list implementation.
type ListStressTest struct {
	Workers  int
	Duration time.Duration
	Timeout  time.Duration
	// LowStress disables membership and connection churn, measuring peer
	// selection baseline performance without interference.
	LowStress bool
	New       func(peer.Transport) peer.ChooserList
}

// Logger is the interface needed by reports to log results.
// The testing.T is an example of a logger.
type Logger interface {
	Logf(format string, args ...interface{})
}

// Log writes the parameters for a stress test.
func (t ListStressTest) Log(logger Logger) {
	logger.Logf("choosers: %d\n", t.Workers)
	logger.Logf("duration: %s\n", t.Duration)
	logger.Logf("timeout:  %s\n", t.Timeout)
}

// Run runs a stress test on a peer list.
//
// The stress test creates a fake transport and a vector of fake peers.
// The test concurrently chooses peers from the list with some number of workers
// while simultaneously adding and removing peers from the peer list and
// simulating connection and disconnection with those peers.
func (t ListStressTest) Run(logger Logger) *ListStressTestReport {
	transport := NewFakeTransport()
	list := t.New(transport)
	report := newStressReport(0)

	s := stressor{
		stop:      make(chan struct{}),
		reports:   make(chan *ListStressTestReport),
		timeout:   t.Timeout,
		transport: transport,
		list:      list,
		logger:    logger,
	}

	if err := s.list.Start(); err != nil {
		s.logger.Logf("list start error: %s\n", err.Error())
	}

	var stressors int
	if t.LowStress {
		for i := uint(0); i < numIds; i++ {
			s.transport.SimulateConnect(bitIds[i])
		}
		err := s.list.Update(peer.ListUpdates{
			Additions: idsForBits(allIdsMask),
		})
		if err != nil {
			s.logger.Logf("list update error: %s\n", err.Error())
			report.Errors++
		}
		report.Updates++
	} else {
		go s.stressTransport(s.reports)
		go s.stressList(s.reports)
		stressors = 2
	}
	for i := 0; i < t.Workers; i++ {
		go s.stressChooser(i)
	}

	time.Sleep(t.Duration)

	close(s.stop)

	for i := 0; i < t.Workers+stressors; i++ {
		report.merge(<-s.reports)
	}

	if err := s.list.Stop(); err != nil {
		s.logger.Logf("list stop error: %s\n", err.Error())
	}

	return report
}

// ListStressTestReport catalogs the results of a peer list stress test.
//
// Each worker keeps track of its own statistics then sends them through
// a channel to the test runner.
// This allows each worker to have independent memory for its log reports and
// reduces the need for synchronization across threads, which could interfere
// with the test.
// The reports get merged into a final report.
type ListStressTestReport struct {
	Workers int
	Errors  int
	Choices int
	Updates int
	Min     time.Duration
	Max     time.Duration
	Total   time.Duration
}

func newStressReport(numWorkers int) *ListStressTestReport {
	return &ListStressTestReport{
		Workers: numWorkers,
		Min:     1000 * time.Second,
	}
}

// Log writes the vital statistics for a stress test.
func (r *ListStressTestReport) Log(logger Logger) {
	logger.Logf("choices:  %d\n", r.Choices)
	logger.Logf("updates:  %d\n", r.Updates)
	logger.Logf("errors:   %d\n", r.Errors)
	logger.Logf("min:      %s\n", r.Min)
	if r.Choices != 0 {
		logger.Logf("mean:     %s\n", r.Total/time.Duration(r.Choices))
	}
	logger.Logf("max:      %s\n", r.Max)
}

// add tracks the latency for a choice of a particular peer.
// the idIndex refers to the peer that was selected.
// in a future version of this test, we can use this id index to show which
// peers were favored by a peer list’s strategy over time.
func (r *ListStressTestReport) add(idIndex int, dur time.Duration) {
	r.Choices++
	r.Min = min(r.Min, dur)
	r.Max = max(r.Max, dur)
	r.Total += dur
}

// merge merges test reports from independent workers.
func (r *ListStressTestReport) merge(s *ListStressTestReport) {
	r.Workers += s.Workers
	r.Errors += s.Errors
	r.Choices += s.Choices
	r.Updates += s.Updates
	r.Min = min(r.Min, s.Min)
	r.Max = max(r.Max, s.Max)
	r.Total += s.Total
}

// stressor tracks the parameters and state for a single stress test worker.
type stressor struct {
	// stop closed to signal all workers to stop.
	stop chan struct{}
	// reports is the channel to which the final report must be sent to singal
	// that the worker goroutine is done and transfer ownership of the report
	// memory to the test for merging.
	reports   chan *ListStressTestReport
	timeout   time.Duration
	transport *FakeTransport
	list      peer.ChooserList
	logger    Logger
}

// stressTransport randomly connects and disconnects each of the 63 known peers.
// These peers may or may not be retained by the peer list at the time the
// connection status changes.
func (s *stressor) stressTransport(reports chan<- *ListStressTestReport) {
	report := newStressReport(0)
	rng := rand.NewSource(0)

	_ = s.transport.Start()
	defer func() {
		_ = s.transport.Stop()
	}()

	// Until we receive a signal to stop...
Loop:
	for {
		select {
		case <-s.stop:
			break Loop
		default:
		}

		// Construt a random bit vector, where each bit signifies whether the
		// peer for that index should be connected or disconnected.
		bits := rng.Int63()
		// A consequence of this is that we may send connected notifications to
		// peers that are already connected, etc.
		// These are valid cases to exercise in a stress test, even if they are
		// not desirable behaviors of a real transport.
		for i := uint(0); i < numIds; i++ {
			bit := (1 << i) & bits
			if bit != 0 {
				s.transport.SimulateConnect(bitIds[i])
			} else {
				s.transport.SimulateDisconnect(bitIds[i])
			}
		}
	}

	reports <- report
}

// stressList sends membership changes to a peer list, using a random subset of all 63 peers every time.
// Each change will tend to include half of the peers, tend to remove a quarter
// from the previous round and add a quarter of the peers for the next round.
// As above, we track whether the peer list has each peer using a bit vector,
// so we can easily use bitwise operations for set differences (&^) and all of
// the identifiers are interned up front to avoid allocations.
// This allows us to send peer list updates very quickly.
func (s *stressor) stressList(reports chan<- *ListStressTestReport) {
	report := newStressReport(0)
	rng := rand.NewSource(1)
	var oldBits int64

	// Until we are asked to stop...
Loop:
	for {
		select {
		case <-s.stop:
			break Loop
		default:
		}

		// Construct peer list updates by giving every peer a 50/50 chance of
		// being included in each round.
		// Use set difference bitwise operations to construct the lists of
		// identifiers to add and remove from the current and previous bit
		// vectors.
		newBits := rng.Int63()
		additions := idsForBits(newBits &^ oldBits)
		removals := idsForBits(oldBits &^ newBits)
		err := s.list.Update(peer.ListUpdates{
			Additions: additions,
			Removals:  removals,
		})
		if err != nil {
			s.logger.Logf("list update error: %s\n", err.Error())
			report.Errors++
			break Loop
		}
		report.Updates++
		oldBits = newBits
	}

	// Clean up.
	err := s.list.Update(peer.ListUpdates{
		Removals: idsForBits(oldBits),
	})
	if err != nil {
		s.logger.Logf("final list update error: %s\n", err.Error())
		report.Errors++
	}

	reports <- report
}

// stressChooser rapidly
func (s *stressor) stressChooser(i int) {
	rng := rand.NewSource(int64(i))
	report := newStressReport(1)

	// Until we are asked to stop...
Loop:
	for {
		// We check for the stop signal before choosing instead of after
		// because the continue statement in the error case bypasses the end of
		// the loop to return here and could cause a deadlock if the other
		// stressors exit first.
		select {
		case <-s.stop:
			break Loop
		default:
		}

		// Request a peer from the peer list.
		// We use a random pre-allocated shard key to exercise the hashring in
		// particular, but this is harmless for all other choosers.
		shardKey := shardKeys[rng.Int63()&shardKeysMask]
		ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
		defer cancel()
		start := time.Now()
		peer, onFinish, err := s.list.Choose(ctx, &transport.Request{ShardKey: shardKey})
		stop := time.Now()
		if err != nil {
			s.logger.Logf("choose error: %s\n", err.Error())
			report.Errors++
			continue
		}
		// This is a good point for a future version to inject varying load
		// based on the identifier of the peer that was selected, to show how
		// each list behaves in the face of variations in speed of individual
		// instances.
		onFinish(nil)
		cancel()

		// Report the latency and identifier of the selected peer.
		id := peer.Identifier()
		index := idIndexes[id]
		report.add(index, stop.Sub(start))
	}

	s.reports <- report
}

// Accessories hereafter.

const (
	// We use a 64 bit vector for peer identifiers, but only get to use 63 bits
	// since the Go random number generator only offers 63 bits of entropy.
	numIds     = 63
	allIdsMask = 1<<numIds - 1
	// We will use 256 unique shard keys.
	shardKeysWidth = 8
	numShardKeys   = 1 << shardKeysWidth
	shardKeysMask  = numShardKeys - 1
)

// pre-allocated vectors for identifiers and shard keys.
var (
	// Each identifier is a string: the name of its own index.
	bitIds [numIds]peer.Identifier
	// Reverse lookup.
	idIndexes map[string]int
	shardKeys [numShardKeys]string
)

func init() {
	idIndexes = make(map[string]int, numIds)
	for i := 0; i < numIds; i++ {
		name := strconv.Itoa(i)
		bitIds[i] = hostport.PeerIdentifier(name)
		idIndexes[name] = i
	}
	for i := 0; i < numShardKeys; i++ {
		shardKeys[i] = strconv.Itoa(i)
	}
}

func idsForBits(bits int64) []peer.Identifier {
	var ids []peer.Identifier
	for i := uint(0); i < numIds; i++ {
		if (1<<i)&bits != 0 {
			ids = append(ids, bitIds[i])
		}
	}
	return ids
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
