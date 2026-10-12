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

package http

import (
	"go.uber.org/atomic"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/zap"
)

// dialerSeq numbers Dialers in logs (TEST BRANCH).
var dialerSeq = atomic.NewInt64(0)

// NewDialer creates a transport that is decorated to optionally isolate
// retained peers from other dialers, and to apply per-outbound options (such
// as OutboundConnectionPool) to every peer it retains.
func (t *Transport) NewDialer(options ...DialOption) *Dialer {
	d := &Dialer{trans: t, options: newDialOptions(options), dialerID: dialerSeq.Inc()}
	t.logger.Info("http2: [test] dialer created",
		zap.Int64("dialerID", d.dialerID), zap.Int("dialOptions", len(options)))
	return d
}

// Dialer is a decorator for an HTTP transport that can isolate its peers,
// and therefore connections, from other dialers, and that threads dial
// options through to every retained peer.
type Dialer struct {
	trans           *Transport
	options         *dialOptions
	connectionScope *connectionScope
	// dialerID identifies this Dialer in logs.
	dialerID int64
}

var _ peer.Transport = (*Dialer)(nil)

// WithConnectionIsolation returns a copy of the Dialer whose peers, and
// therefore connections or connection pools, are not shared with other
// dialers retaining the same address.
//
// Callers should create one isolated Dialer per logical outbound. Requests
// within that outbound continue to share peers normally.
//
// OutboundConnectionPool on this Dialer applies to peers retained through it.
// Isolation is what keeps those peers (and their pools) from being shared
// with other dialers. Live outbound providers are attached at YAML
// buildOutbound from WithOutboundLiveConnectionPoolProvider.
func (d *Dialer) WithConnectionIsolation() *Dialer {
	isolated := *d
	optionsCopy := *d.options
	isolated.options = &optionsCopy
	isolated.connectionScope = new(connectionScope)
	isolated.dialerID = dialerSeq.Inc()
	d.trans.logger.Info("http2: [test] isolated dialer created",
		zap.Int64("dialerID", isolated.dialerID),
		zap.Int64("derivedFromDialerID", d.dialerID),
		zap.String("connectionScope", scopeLabel(isolated.connectionScope)),
		zap.Bool("poolOverride", isolated.options.connPoolOverride != nil),
		zap.Bool("liveProvider", isolated.options.poolConfigProvider != nil))
	return &isolated
}

// outboundPeerTransport returns the peer.Transport a new outbound built by this
// Transport retains its peers through: the Transport itself, or, with
// IsolateConnectionsPerOutbound, a Dialer with a connection scope of its own so
// that no other outbound shares the outbound's peers. Call it once per outbound.
func (t *Transport) outboundPeerTransport() peer.Transport {
	if t.isolateOutboundConnections {
		return t.NewDialer().WithConnectionIsolation()
	}
	return t
}

// RetainPeer retains the identified peer.
func (d *Dialer) RetainPeer(id peer.Identifier, ps peer.Subscriber) (peer.Peer, error) {
	d.trans.logger.Info("http2: [test] dialer retaining peer",
		zap.Int64("dialerID", d.dialerID),
		zap.String("peerIdentifier", id.Identifier()),
		zap.String("connectionScope", scopeLabel(d.connectionScope)))
	return d.trans.retainPeer(id, d.options, d.connectionScope, ps)
}

// ReleasePeer releases the identified peer.
func (d *Dialer) ReleasePeer(id peer.Identifier, ps peer.Subscriber) error {
	d.trans.logger.Info("http2: [test] dialer releasing peer",
		zap.Int64("dialerID", d.dialerID),
		zap.String("peerIdentifier", id.Identifier()),
		zap.String("connectionScope", scopeLabel(d.connectionScope)))
	return d.trans.releasePeer(id, d.connectionScope, ps)
}
