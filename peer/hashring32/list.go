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

// package harshing32yarpc provides peerlist bindings for hashring32 in YARPC.

import (
	"context"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/x/introspection"
	"go.uber.org/yarpc/peer/abstractlist"
	"go.uber.org/yarpc/peer/hashring32/internal/hashring32"
	"go.uber.org/zap"
)

type options struct {
	offsetHeader            string
	peerOverrideHeader      string
	alternateShardKeyHeader string
	peerRingOptions         []hashring32.Option
	defaultChooseTimeout    *time.Duration
	logger                  *zap.Logger
}

// Option customizes the behavior of hashring32 peer list.
type Option interface {
	apply(*options)
}

// OffsetHeader is the option function that allows a user to pass a header
// key to the hashring to adjust to N value in the hashring choose function.
func OffsetHeader(offsetHeader string) Option {
	return offsetHeaderOption{offsetHeader: offsetHeader}
}

type offsetHeaderOption struct {
	offsetHeader string
}

func (o offsetHeaderOption) apply(opts *options) {
	opts.offsetHeader = o.offsetHeader
}

// PeerOverrideHeader allows clients to pass a header containing the shard
// identifier for a specific peer to override the destination address for the
// outgoing request.
//
// For example, if the peer list uses addresses to identify peers, the hash
// ring will have retained a peer for every known address.
// Specifying an address like "127.0.0.1" in the route override header will
// deflect the request to that exact peer.
// If that peer is not available, the request will continue on to the peer
// implied by the shard key.
func PeerOverrideHeader(peerOverrideHeader string) Option {
	return peerOverrideHeaderOption{peerOverrideHeader: peerOverrideHeader}
}

type peerOverrideHeaderOption struct {
	peerOverrideHeader string
}

func (o peerOverrideHeaderOption) apply(opts *options) {
	opts.peerOverrideHeader = o.peerOverrideHeader
}

// AlternateShardKeyHeader allows clients to pass a header containing the shard
// identifier for a specific peer to override the destination address for the
// outgoing request.
func AlternateShardKeyHeader(alternateShardKeyHeader string) Option {
	return alternateShardKeyHeaderOption{alternateShardKeyHeader: alternateShardKeyHeader}
}

type alternateShardKeyHeaderOption struct {
	alternateShardKeyHeader string
}

func (o alternateShardKeyHeaderOption) apply(opts *options) {
	opts.alternateShardKeyHeader = o.alternateShardKeyHeader
}

// ReplicaDelimiter overrides the the delimiter the hash ring uses to construct
// replica identifiers from peer identifiers and replica numbers.
//
// The default delimiter is an empty string.
func ReplicaDelimiter(delimiter string) Option {
	return replicaDelimiterOption{delimiter: delimiter}
}

type replicaDelimiterOption struct {
	delimiter string
}

func (o replicaDelimiterOption) apply(opts *options) {
	opts.peerRingOptions = append(
		opts.peerRingOptions,
		hashring32.ReplicaFormatter(
			hashring32.DelimitedReplicaFormatter(o.delimiter),
		),
	)
}

// Logger threads a logger into the hash ring constructor.
func Logger(logger *zap.Logger) Option {
	return loggerOption{logger: logger}
}

type loggerOption struct {
	logger *zap.Logger
}

func (o loggerOption) apply(opts *options) {
	opts.logger = o.logger
}

// NumReplicas allos client to specify the number of replicas to use for each peer.
//
// More replicas produces a more even distribution of entities and slower
// membership updates.
func NumReplicas(n int) Option {
	return numReplicasOption{numReplicas: n}
}

type numReplicasOption struct {
	numReplicas int
}

func (n numReplicasOption) apply(opts *options) {
	opts.peerRingOptions = append(
		opts.peerRingOptions,
		hashring32.NumReplicas(
			n.numReplicas,
		),
	)
}

// NumPeersEstimate allows client to specifiy an estimate for the number of identified peers
// the hashring will contain.
func NumPeersEstimate(n int) Option {
	return numPeersEstimateOption{numPeersEstimate: n}
}

type numPeersEstimateOption struct {
	numPeersEstimate int
}

func (n numPeersEstimateOption) apply(opts *options) {
	opts.peerRingOptions = append(
		opts.peerRingOptions,
		hashring32.NumPeersEstimate(
			n.numPeersEstimate,
		),
	)
}

// DefaultChooseTimeout specifies the default timeout to add to 'Choose' calls
// without context deadlines. This prevents long-lived streams from setting
// calling deadlines.
//
// Defaults to 500ms.
func DefaultChooseTimeout(timeout time.Duration) Option {
	return optionFunc(func(options *options) {
		options.defaultChooseTimeout = &timeout
	})
}

type optionFunc func(*options)

func (f optionFunc) apply(options *options) { f(options) }

// New creates a new hashring32 peer list.
func New(transport peer.Transport, hashFunc hashring32.HashFunc32, opts ...Option) *List {
	var options options
	for _, o := range opts {
		o.apply(&options)
	}

	logger := options.logger
	if logger == nil {
		logger = zap.NewNop()
	}

	ring := newPeerRing(
		hashFunc,
		options.offsetHeader,
		options.peerOverrideHeader,
		options.alternateShardKeyHeader,
		logger,
		options.peerRingOptions...,
	)

	plOpts := []abstractlist.Option{abstractlist.Logger(logger)}

	if options.defaultChooseTimeout != nil {
		plOpts = append(plOpts, abstractlist.DefaultChooseTimeout(*options.defaultChooseTimeout))
	}

	return &List{
		list: abstractlist.New("hashring32", transport, ring, plOpts...),
	}
}

// List is a PeerList which chooses peers based on a hashing function.
type List struct {
	list *abstractlist.List
}

// Start causes the peer list to start.
//
// Starting will retain all peers that have been added but not removed
// the first time it is called.
//
// Start may be called any number of times and in any order in relation to Stop
// but will only cause the list to start the first time, and only if it has not
// already been stopped.
func (l *List) Start() error {
	return l.list.Start()
}

// Stop causes the peer list to stop.
//
// Stopping will release all retained peers to the underlying transport.
//
// Stop may be called any number of times and in order in relation to Start but
// will only cause the list to stop the first time, and only if it has
// previously been started.
func (l *List) Stop() error {
	return l.list.Stop()
}

// IsRunning returns whether the list has started and not yet stopped.
func (l *List) IsRunning() bool {
	return l.list.IsRunning()
}

// Choose returns a peer, suitable for sending a request.
//
// The peer is not guaranteed to be connected and available, but the peer list
// makes every attempt to ensure this and minimize the probability that a
// chosen peer will fail to carry a request.
func (l *List) Choose(ctx context.Context, req *transport.Request) (peer peer.Peer, onFinish func(error), err error) {
	return l.list.Choose(ctx, req)
}

// Update may add and remove logical peers in the list.
//
// The peer list uses a transport to obtain a physical peer for each logical
// peer.
// The transport is responsible for informing the peer list whether the peer is
// available or unavailable, but cannot guarantee that the peer will still be
// available after it is chosen.
func (l *List) Update(updates peer.ListUpdates) error {
	return l.list.Update(updates)
}

// NotifyStatusChanged forwards a status change notification to an individual
// peer in the list.
//
// This satisfies the peer.Subscriber interface and should only be used to
// send notifications in tests.
// The list's RetainPeer and ReleasePeer methods deal with an individual
// peer.Subscriber instance for each peer in the list, avoiding a map lookup.
func (l *List) NotifyStatusChanged(pid peer.Identifier) {
	l.list.NotifyStatusChanged(pid)
}

// Introspect reveals information about the list to the internal YARPC
// introspection system.
func (l *List) Introspect() introspection.ChooserStatus {
	return l.list.Introspect()
}

// Peers produces a slice of all retained peers.
func (l *List) Peers() []peer.StatusPeer {
	return l.list.Peers()
}
