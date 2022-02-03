// Copyright (c) 2022 Uber Technologies, Inc.
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

package hostport

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/peer"
	. "go.uber.org/yarpc/api/peer/peertest"
)

func TestPeerIdentifier(t *testing.T) {
	tests := []struct {
		hostport           string
		expectedIdentifier string
	}{
		{
			"localhost:12345",
			"localhost:12345",
		},
		{
			"123.123.123.123:12345",
			"123.123.123.123:12345",
		},
	}

	for _, tt := range tests {
		pi := PeerIdentifier(tt.hostport)

		assert.Equal(t, tt.expectedIdentifier, pi.Identifier())
	}
}

func TestPeer(t *testing.T) {
	type testStruct struct {
		msg string

		// PeerID for all the hostport.Peer initialization
		inputPeerID string

		// List of "Subscribers" that we can define beforehand and reference in our PeerActions
		// using the "ID" field in the subscriber definition, the "ExpectedNotifyCount" is the
		// expected number of times the subscriber will be notified
		SubDefinitions []SubscriberDefinition

		// List of actions to be applied to the peer
		actions []PeerAction

		// Expected result from running Identifier() on the peer after the actions have been applied
		expectedIdentifier string

		// Expected result from running HostPort() on the peer after the actions have been applied
		expectedHostPort string

		// Expected result from running Status() on the peer after the actions have been applied
		expectedStatus peer.Status

		// Expected subscribers in the Peer's "subscribers" map after the actions have been applied
		expectedSubscribers []string
	}
	tests := []testStruct{
		{
			msg:                "create",
			inputPeerID:        "localhost:12345",
			expectedIdentifier: "localhost:12345",
			expectedHostPort:   "localhost:12345",
			expectedStatus: peer.Status{
				PendingRequestCount: 0,
				ConnectionStatus:    peer.Unavailable,
			},
		},
		{
			msg:            "start request",
			SubDefinitions: []SubscriberDefinition{{ID: "1", ExpectedNotifyCount: 1}},
			actions: []PeerAction{
				SubscribeAction{SubscriberID: "1", ExpectedSubCount: 1},
				StartStopReqAction{Stop: false},
			},
			expectedSubscribers: []string{"1"},
			expectedStatus: peer.Status{
				PendingRequestCount: 1,
				ConnectionStatus:    peer.Unavailable,
			},
		},
		{
			msg:            "start request stop request",
			SubDefinitions: []SubscriberDefinition{{ID: "1", ExpectedNotifyCount: 2}},
			actions: []PeerAction{
				SubscribeAction{SubscriberID: "1", ExpectedSubCount: 1},
				StartStopReqAction{Stop: true},
			},
			expectedSubscribers: []string{"1"},
			expectedStatus: peer.Status{
				PendingRequestCount: 0,
				ConnectionStatus:    peer.Unavailable,
			},
		},
		{
			msg:            "start 5 stop 2",
			SubDefinitions: []SubscriberDefinition{{ID: "1", ExpectedNotifyCount: 7}},
			actions: []PeerAction{
				SubscribeAction{SubscriberID: "1", ExpectedSubCount: 1},
				StartStopReqAction{Stop: true},
				StartStopReqAction{Stop: false},
				StartStopReqAction{Stop: false},
				StartStopReqAction{Stop: true},
				StartStopReqAction{Stop: false},
			},
			expectedSubscribers: []string{"1"},
			expectedStatus: peer.Status{
				PendingRequestCount: 3,
				ConnectionStatus:    peer.Unavailable,
			},
		},
		{
			msg:            "start 5 stop 5",
			SubDefinitions: []SubscriberDefinition{{ID: "1", ExpectedNotifyCount: 10}},
			actions: []PeerAction{
				SubscribeAction{SubscriberID: "1", ExpectedSubCount: 1},
				StartStopReqAction{Stop: true},
				StartStopReqAction{Stop: true},
				StartStopReqAction{Stop: true},
				StartStopReqAction{Stop: true},
				StartStopReqAction{Stop: true},
			},
			expectedSubscribers: []string{"1"},
			expectedStatus: peer.Status{

				ConnectionStatus: peer.Unavailable,
			},
		},
		{
			msg: "set status",
			SubDefinitions: []SubscriberDefinition{
				{ID: "1", ExpectedNotifyCount: 1},
				{ID: "2", ExpectedNotifyCount: 1},
				{ID: "3", ExpectedNotifyCount: 1},
			},
			actions: []PeerAction{
				SubscribeAction{SubscriberID: "1", ExpectedSubCount: 1},
				SubscribeAction{SubscriberID: "2", ExpectedSubCount: 2},
				SubscribeAction{SubscriberID: "3", ExpectedSubCount: 3},
				SetStatusAction{InputStatus: peer.Available},
			},
			expectedSubscribers: []string{"1", "2", "3"},
			expectedStatus: peer.Status{
				PendingRequestCount: 0,
				ConnectionStatus:    peer.Available,
			},
		},
		{
			msg: "incremental subscribe",
			SubDefinitions: []SubscriberDefinition{
				{ID: "1", ExpectedNotifyCount: 3},
				{ID: "2", ExpectedNotifyCount: 2},
				{ID: "3", ExpectedNotifyCount: 1},
			},
			actions: []PeerAction{
				SubscribeAction{SubscriberID: "1", ExpectedSubCount: 1},
				SetStatusAction{InputStatus: peer.Available},
				SubscribeAction{SubscriberID: "2", ExpectedSubCount: 2},
				SetStatusAction{InputStatus: peer.Available},
				SubscribeAction{SubscriberID: "3", ExpectedSubCount: 3},
				SetStatusAction{InputStatus: peer.Available},
			},
			expectedSubscribers: []string{"1", "2", "3"},
			expectedStatus: peer.Status{
				PendingRequestCount: 0,
				ConnectionStatus:    peer.Available,
			},
		},
		{
			msg: "subscribe unsubscribe",
			SubDefinitions: []SubscriberDefinition{
				{ID: "1", ExpectedNotifyCount: 1},
			},
			actions: []PeerAction{
				SubscribeAction{SubscriberID: "1", ExpectedSubCount: 1},
				SetStatusAction{InputStatus: peer.Available},
				UnsubscribeAction{SubscriberID: "1", ExpectedSubCount: 0},
				SetStatusAction{InputStatus: peer.Available},
			},
			expectedStatus: peer.Status{
				PendingRequestCount: 0,
				ConnectionStatus:    peer.Available,
			},
		},
		{
			msg: "incremental subscribe unsubscribe",
			SubDefinitions: []SubscriberDefinition{
				{ID: "1", ExpectedNotifyCount: 5},
				{ID: "2", ExpectedNotifyCount: 3},
				{ID: "3", ExpectedNotifyCount: 1},
			},
			actions: []PeerAction{
				SubscribeAction{SubscriberID: "1", ExpectedSubCount: 1},
				SetStatusAction{InputStatus: peer.Available},
				SubscribeAction{SubscriberID: "2", ExpectedSubCount: 2},
				SetStatusAction{InputStatus: peer.Available},
				SubscribeAction{SubscriberID: "3", ExpectedSubCount: 3},
				SetStatusAction{InputStatus: peer.Available},
				UnsubscribeAction{SubscriberID: "3", ExpectedSubCount: 2},
				SetStatusAction{InputStatus: peer.Available},
				UnsubscribeAction{SubscriberID: "2", ExpectedSubCount: 1},
				SetStatusAction{InputStatus: peer.Available},
				UnsubscribeAction{SubscriberID: "1", ExpectedSubCount: 0},
				SetStatusAction{InputStatus: peer.Available},
			},
			expectedStatus: peer.Status{
				PendingRequestCount: 0,
				ConnectionStatus:    peer.Available,
			},
		},
		{
			msg: "unsubscribe error",
			SubDefinitions: []SubscriberDefinition{
				{ID: "1", ExpectedNotifyCount: 0},
			},
			actions: []PeerAction{
				UnsubscribeAction{
					SubscriberID:     "1",
					ExpectedErrType:  peer.ErrPeerHasNoReferenceToSubscriber{},
					ExpectedSubCount: 0,
				},
			},
			expectedStatus: peer.Status{
				PendingRequestCount: 0,
				ConnectionStatus:    peer.Unavailable,
			},
		},
		{
			msg: "concurrent update and subscribe",
			SubDefinitions: []SubscriberDefinition{
				{ID: "1", ExpectedNotifyCount: 2},
			},
			actions: []PeerAction{
				PeerConcurrentAction{
					Actions: []PeerAction{
						SubscribeAction{SubscriberID: "1", ExpectedSubCount: 1},
						SetStatusAction{InputStatus: peer.Available},
						SetStatusAction{InputStatus: peer.Available},
					},
					Wait: time.Millisecond * 70,
				},
			},
			expectedSubscribers: []string{"1"},
			expectedStatus: peer.Status{
				PendingRequestCount: 0,
				ConnectionStatus:    peer.Available,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			if tt.inputPeerID == "" {
				tt.inputPeerID = "localhost:12345"
				tt.expectedIdentifier = "localhost:12345"
				tt.expectedHostPort = "localhost:12345"
			}

			transport := NewMockTransport(mockCtrl)

			peer := NewPeer(PeerIdentifier(tt.inputPeerID), transport)

			deps := &Dependencies{
				Subscribers: CreateSubscriberMap(mockCtrl, tt.SubDefinitions),
			}

			ApplyPeerActions(t, peer, tt.actions, deps)

			assert.Equal(t, tt.expectedIdentifier, peer.Identifier())
			assert.Equal(t, tt.expectedHostPort, peer.HostPort())
			assert.Equal(t, transport, peer.Transport())
			assert.Equal(t, tt.expectedStatus, peer.Status())

			assert.Len(t, peer.subscribers, len(tt.expectedSubscribers))
			for _, subID := range tt.expectedSubscribers {
				sub, ok := deps.Subscribers[subID]
				assert.True(t, ok, "referenced subscriber id that does not exist %s", sub)

				_, ok = peer.subscribers[sub]
				assert.True(t, ok, "peer did not have reference to subscriber %v", sub)
			}
		})
	}
}
