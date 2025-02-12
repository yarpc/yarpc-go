// Copyright (c) 2025 Uber Technologies, Inc.
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

package abstractpeer

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/internal/testtime"
)

// There are no actual tests in this file, it contains a series of helper methods
// for testing abstractpeer.Peers

// Dependences are passed through all the PeerActions in order to pass certain
// state in between Actions
type Dependencies struct {
	Subscribers map[string]peer.Subscriber
}

// PeerAction defines actions that can be applied on a abstractpeer.Peer
type PeerAction interface {
	Apply(*testing.T, *Peer, *Dependencies)
}

// StartStopReqAction will run a StartRequest and (optionally) EndRequest
type StartStopReqAction struct {
	Stop bool
}

// Apply will run StartRequest and (optionally) EndRequest
func (sa StartStopReqAction) Apply(t *testing.T, p *Peer, d *Dependencies) {
	p.StartRequest()
	p.NotifyStatusChanged()
	if sa.Stop {
		p.EndRequest()
		p.NotifyStatusChanged()
	}
}

// SetStatusAction will run a SetStatus on a Peer
type SetStatusAction struct {
	InputStatus peer.ConnectionStatus
}

// Apply will run SetStatus on the Peer
func (sa SetStatusAction) Apply(t *testing.T, p *Peer, d *Dependencies) {
	p.SetStatus(sa.InputStatus)
	p.NotifyStatusChanged()

	assert.Equal(t, sa.InputStatus, p.Status().ConnectionStatus)
}

// SubscribeAction will run an Subscribe on a Peer
type SubscribeAction struct {
	// SubscriberID is a unique identifier for a subscriber that is
	// contained in the Dependencies object passed in Apply
	SubscriberID string

	// ExpectedSubCount is the number of subscribers on the Peer after
	// the subscription
	ExpectedSubCount int
}

// Apply will run Subscribe on a Peer
func (sa SubscribeAction) Apply(t *testing.T, p *Peer, d *Dependencies) {
	sub, ok := d.Subscribers[sa.SubscriberID]
	assert.True(t, ok, "referenced a subscriberID that does not exist %s", sa.SubscriberID)

	p.Subscribe(sub)

	assert.Equal(t, sa.ExpectedSubCount, p.NumSubscribers())
}

// UnsubscribeAction will run Unsubscribe on a Peer
type UnsubscribeAction struct {
	// SubscriberID is a unique identifier for a subscriber that is
	// contained in the Dependencies object passed in Apply
	SubscriberID string

	// ExpectedErrType is the type of error that is expected to be returned
	// from Unsubscribe
	ExpectedErrType error

	// ExpectedSubCount is the number of subscribers on the Peer after
	// the subscription
	ExpectedSubCount int
}

// Apply will run Unsubscribe from the Peer and assert on the result
func (ua UnsubscribeAction) Apply(t *testing.T, p *Peer, d *Dependencies) {
	sub, ok := d.Subscribers[ua.SubscriberID]
	assert.True(t, ok, "referenced a subscriberID that does not exist %s", ua.SubscriberID)

	err := p.Unsubscribe(sub)

	assert.Equal(t, ua.ExpectedSubCount, p.NumSubscribers())
	if err != nil {
		assert.IsType(t, ua.ExpectedErrType, err)
	} else {
		assert.Nil(t, ua.ExpectedErrType)
	}
}

// PeerConcurrentAction will run a series of actions in parallel
type PeerConcurrentAction struct {
	Actions []PeerAction
	Wait    time.Duration
}

// Apply runs all the ConcurrentAction's actions in goroutines with a delay of `Wait`
// between each action. Returns when all actions have finished executing
func (a PeerConcurrentAction) Apply(t *testing.T, p *Peer, d *Dependencies) {
	var wg sync.WaitGroup

	wg.Add(len(a.Actions))
	for _, action := range a.Actions {
		go func(ac PeerAction) {
			defer wg.Done()
			ac.Apply(t, p, d)
		}(action)

		if a.Wait > 0 {
			testtime.Sleep(a.Wait)
		}
	}

	wg.Wait()
}

// ApplyPeerActions runs all the PeerActions on the Peer
func ApplyPeerActions(t *testing.T, p *Peer, actions []PeerAction, d *Dependencies) {
	for i, action := range actions {
		t.Run(fmt.Sprintf("action #%d: %T", i, action), func(t *testing.T) {
			action.Apply(t, p, d)
		})
	}
}
