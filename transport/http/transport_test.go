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
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	. "go.uber.org/yarpc/api/peer/peertest"
	"go.uber.org/yarpc/internal/testtime"
	ypeer "go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
)

// NoJitter is a transport option only available in tests, to disable jitter
// between connection attempts.
func NoJitter() TransportOption {
	return func(options *transportOptions) {
		options.jitter = func(n int64) int64 {
			return n
		}
	}
}

type peerExpectation struct {
	id          string
	subscribers []string
}

func createPeerIdentifierMap(ids []string) map[string]peer.Identifier {
	pids := make(map[string]peer.Identifier, len(ids))
	for _, id := range ids {
		pids[id] = &testIdentifier{id}
	}
	return pids
}

func TestTransport(t *testing.T) {
	type testStruct struct {
		msg string

		// identifiers defines all the Identifiers that will be used in
		// the actions up from so they can be generated and passed as deps
		identifiers []string

		// subscriberDefs defines all the Subscribers that will be used in
		// the actions up from so they can be generated and passed as deps
		subscriberDefs []SubscriberDefinition

		// actions are the actions that will be applied against the transport
		actions []TransportAction

		// expectedPeers are a list of peers (and those peer's subscribers)
		// that are expected on the transport after the actions
		expectedPeers []peerExpectation
	}
	tests := []testStruct{
		{
			msg:         "one retain",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
			},
			actions: []TransportAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1"}},
			},
		},
		{
			msg:         "one retain one release",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
			},
			actions: []TransportAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s1"},
			},
		},
		{
			msg:         "three retains",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2"},
				{ID: "s3"},
			},
			actions: []TransportAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s2", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s3", ExpectedPeerID: "i1"},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1", "s2", "s3"}},
			},
		},
		{
			msg:         "three retains one release",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2r"},
				{ID: "s3"},
			},
			actions: []TransportAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s2r", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s3", ExpectedPeerID: "i1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s2r"},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1", "s3"}},
			},
		},
		{
			msg:         "three retains, three release",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2"},
				{ID: "s3"},
			},
			actions: []TransportAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s2", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s3", ExpectedPeerID: "i1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s2"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s3"},
			},
		},
		{
			msg:         "no retains one release",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
			},
			actions: []TransportAction{
				ReleaseAction{
					InputIdentifierID: "i1",
					InputSubscriberID: "s1",
					ExpectedErrType:   peer.ErrTransportHasNoReferenceToPeer{},
				},
			},
		},
		{
			msg:         "one retains, one release (from different subscriber)",
			identifiers: []string{"i1"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2"},
			},
			actions: []TransportAction{
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				ReleaseAction{
					InputIdentifierID: "i1",
					InputSubscriberID: "s2",
					ExpectedErrType:   peer.ErrPeerHasNoReferenceToSubscriber{},
				},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1"}},
			},
		},
		{
			msg:         "multi peer retain/release",
			identifiers: []string{"i1", "i2", "i3", "i4r", "i5r"},
			subscriberDefs: []SubscriberDefinition{
				{ID: "s1"},
				{ID: "s2"},
				{ID: "s3"},
				{ID: "s4"},
				{ID: "s5rnd"},
				{ID: "s6rnd"},
				{ID: "s7rnd"},
			},
			actions: []TransportAction{
				// Retains/Releases of i1 (Retain/Release the random peers at the end)
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s5rnd", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s6rnd", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s1", ExpectedPeerID: "i1"},
				RetainAction{InputIdentifierID: "i1", InputSubscriberID: "s2", ExpectedPeerID: "i1"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s5rnd"},
				ReleaseAction{InputIdentifierID: "i1", InputSubscriberID: "s6rnd"},

				// Retains/Releases of i2 (Retain then Release then Retain again)
				RetainAction{InputIdentifierID: "i2", InputSubscriberID: "s2", ExpectedPeerID: "i2"},
				RetainAction{InputIdentifierID: "i2", InputSubscriberID: "s3", ExpectedPeerID: "i2"},
				ReleaseAction{InputIdentifierID: "i2", InputSubscriberID: "s2"},
				ReleaseAction{InputIdentifierID: "i2", InputSubscriberID: "s3"},
				RetainAction{InputIdentifierID: "i2", InputSubscriberID: "s2", ExpectedPeerID: "i2"},
				RetainAction{InputIdentifierID: "i2", InputSubscriberID: "s3", ExpectedPeerID: "i2"},

				// Retains/Releases of i3 (Retain/Release unrelated sub, then retain two)
				RetainAction{InputIdentifierID: "i3", InputSubscriberID: "s7rnd", ExpectedPeerID: "i3"},
				ReleaseAction{InputIdentifierID: "i3", InputSubscriberID: "s7rnd"},
				RetainAction{InputIdentifierID: "i3", InputSubscriberID: "s3", ExpectedPeerID: "i3"},
				RetainAction{InputIdentifierID: "i3", InputSubscriberID: "s4", ExpectedPeerID: "i3"},

				// Retain/Release i4r on random subscriber
				RetainAction{InputIdentifierID: "i4r", InputSubscriberID: "s5rnd", ExpectedPeerID: "i4r"},
				ReleaseAction{InputIdentifierID: "i4r", InputSubscriberID: "s5rnd"},

				// Retain/Release i5r on already used subscriber
				RetainAction{InputIdentifierID: "i5r", InputSubscriberID: "s3", ExpectedPeerID: "i5r"},
				ReleaseAction{InputIdentifierID: "i5r", InputSubscriberID: "s3"},
			},
			expectedPeers: []peerExpectation{
				{id: "i1", subscribers: []string{"s1", "s2"}},
				{id: "i2", subscribers: []string{"s2", "s3"}},
				{id: "i3", subscribers: []string{"s3", "s4"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			transport := NewTransport()
			defer transport.Stop()

			deps := TransportDeps{
				PeerIdentifiers: createPeerIdentifierMap(tt.identifiers),
				Subscribers:     CreateSubscriberMap(mockCtrl, tt.subscriberDefs),
			}
			ApplyTransportActions(t, transport, tt.actions, deps)

			assert.Len(t, transport.peers, len(tt.expectedPeers))
			for _, expectedPeerNode := range tt.expectedPeers {
				p, ok := transport.peers[expectedPeerNode.id]
				assert.True(t, ok)

				if assert.NotNil(t, p) {
					assert.Equal(t, expectedPeerNode.id, p.Identifier())

					// We can't look at the hostport subscribers directly so we'll
					// attempt to remove subscribers and be sure that it doesn't error
					assert.Len(t, expectedPeerNode.subscribers, p.NumSubscribers())
					for _, sub := range expectedPeerNode.subscribers {
						err := p.Unsubscribe(deps.Subscribers[sub])
						assert.NoError(t, err, "peer %s did not have reference to subscriber %s", p.Identifier(), sub)
					}
				}
			}
		})
	}
}

func TestDefaultTransportInitialisation(t *testing.T) {
	transport := NewTransport()

	assert.NotNil(t, transport.h1Transport)
	assert.NotNil(t, transport.h2Transport)
}

func TestTransportClientOpaqueOptions(t *testing.T) {
	// Unfortunately the KeepAlive is obfuscated in the client, so we can't really
	// assert this worked.
	transport := NewTransport(
		KeepAlive(testtime.Second),
		MaxIdleConns(100),
		MaxIdleConnsPerHost(10),
		IdleConnTimeout(1*time.Second),
		DisableCompression(),
		DisableKeepAlives(),
		ResponseHeaderTimeout(1*time.Second),
	)

	assert.NotNil(t, transport.h1Transport)
	assert.NotNil(t, transport.h2Transport)
}

func TestDialContext(t *testing.T) {
	errMsg := "my custom dialer error message"
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, errors.New(errMsg)
	}

	transport := NewTransport(DialContext(dialContext))

	require.NoError(t, transport.Start())
	defer func() { assert.NoError(t, transport.Stop()) }()

	req, err := http.NewRequest("GET", "http://foo.bar", nil)
	require.NoError(t, err)

	outbound := transport.NewOutbound(ypeer.NewSingle(hostport.Identify("foo"), transport))
	require.NoError(t, outbound.Start())
	defer func() { assert.NoError(t, outbound.Stop()) }()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = outbound.RoundTrip(req.WithContext(ctx))
	require.Error(t, err)
	assert.Contains(t, err.Error(), errMsg)
}

type testIdentifier struct {
	id string
}

func (i testIdentifier) Identifier() string {
	return i.id
}
