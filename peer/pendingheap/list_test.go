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

package pendingheap

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/peer"
	. "go.uber.org/yarpc/api/peer/peertest"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/testtime"
	"go.uber.org/yarpc/internal/whitespace"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/yarpcconfig"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap/zaptest"
)

var (
	_noContextDeadlineError = yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, `"fewest-pending-requests" peer list can't wait for peer without a context deadline`)
)

func newNotRunningError(err string) error {
	return yarpcerrors.FailedPreconditionErrorf(`"fewest-pending-requests" peer list is not running: %s`, err)
}

func newUnavailableError(err error) error {
	return yarpcerrors.UnavailableErrorf(`"fewest-pending-requests" peer list timed out waiting for peer: %s`, err.Error())
}

// InsertionOrder is a test option that yields control over random insertion
// ordering. Each number corresponds to the position to swap the newly inserted
// peer's 'last' value.
//
// The function MUST return a number in [0, numPeers)
func InsertionOrder(f func(numPeers int) int) ListOption {
	return func(c *listConfig) {
		c.nextRand = f
	}
}

// DisableRandomInsertion disables random insertions.
func DisableRandomInsertion() ListOption {
	// avoid swaps by always returning the last index
	return InsertionOrder(func(numPeers int) int { return numPeers - 1 })
}

func nextRandFromSlice(indicies []int) func(int) int {
	i := -1
	return func(_ int) int {
		i++
		return indicies[i]
	}
}

func TestPeerHeapList(t *testing.T) {
	type testStruct struct {
		msg string

		// nextRand is used with the InsertionOrder(...) option. If nil, this defaults
		// to DisableRandomInsertion()
		nextRand func(int) int

		// PeerIDs that will be returned from the transport's OnRetain with "Available" status
		retainedAvailablePeerIDs []string

		// PeerIDs that will be returned from the transport's OnRetain with "Unavailable" status
		retainedUnavailablePeerIDs []string

		// PeerIDs that will be released from the transport
		releasedPeerIDs []string

		// A list of actions that will be applied on the PeerList
		peerListActions []PeerListAction

		// PeerIDs expected to be in the PeerList's "Available" list after the actions have been applied
		expectedAvailablePeers []string

		// PeerIDs expected to be in the PeerList's "Unavailable" list after the actions have been applied
		expectedUnavailablePeers []string

		// Boolean indicating whether the PeerList is "running" after the actions have been applied
		expectedRunning bool
	}
	tests := []testStruct{
		{
			msg:                      "setup",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1"}},
			},
			expectedRunning: true,
		},
		{
			msg:                        "setup with disconnected",
			retainedAvailablePeerIDs:   []string{"1"},
			retainedUnavailablePeerIDs: []string{"2"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1", "2"}},
			},
			expectedAvailablePeers:   []string{"1"},
			expectedUnavailablePeers: []string{"2"},
			expectedRunning:          true,
		},
		{
			msg:                      "start",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1"}},
				ChooseAction{
					ExpectedPeer: "1",
				},
			},
			expectedRunning: true,
		},
		{
			msg:                        "start stop",
			retainedAvailablePeerIDs:   []string{"1", "2", "3", "4", "5", "6"},
			retainedUnavailablePeerIDs: []string{"7", "8", "9"},
			releasedPeerIDs:            []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}},
				StopAction{},
				ChooseAction{
					ExpectedErr:         newNotRunningError("could not wait for instance to start running: current state is \"stopped\""),
					InputContextTimeout: 10 * time.Millisecond,
				},
			},
			expectedRunning: false,
		},
		{
			msg:                      "update, start, and choose",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg:                      "start many and choose",
			retainedAvailablePeerIDs: []string{"1", "2", "3", "4", "5", "6"},
			expectedAvailablePeers:   []string{"1", "2", "3", "4", "5", "6"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1", "2", "3", "4", "5", "6"}},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "3"},
				ChooseAction{ExpectedPeer: "4"},
				ChooseAction{ExpectedPeer: "5"},
				ChooseAction{ExpectedPeer: "6"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg:                      "assure start is idempotent",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				StartAction{},
				ChooseAction{
					ExpectedPeer: "1",
				},
			},
			expectedRunning: true,
		},
		{
			msg:                      "stop no start",
			retainedAvailablePeerIDs: []string{},
			releasedPeerIDs:          []string{},
			peerListActions: []PeerListAction{
				StopAction{},
				UpdateAction{AddedPeerIDs: []string{"1"}},
			},
			expectedRunning: false,
		},
		{
			msg: "choose before start",
			peerListActions: []PeerListAction{
				ChooseAction{
					ExpectedErr:         newNotRunningError("context finished while waiting for instance to start: context deadline exceeded"),
					InputContextTimeout: 10 * time.Millisecond,
				},
				ChooseAction{
					ExpectedErr:         newNotRunningError("context finished while waiting for instance to start: context deadline exceeded"),
					InputContextTimeout: 10 * time.Millisecond,
				},
			},
			expectedRunning: false,
		},
		{
			msg:                      "update before start",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				ConcurrentAction{
					Actions: []PeerListAction{
						UpdateAction{AddedPeerIDs: []string{"1"}},
						StartAction{},
					},
					Wait: 20 * time.Millisecond,
				},
			},
			expectedRunning: true,
		},
		{
			msg: "start choose no peers",
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{
					InputContextTimeout: 20 * time.Millisecond,
					ExpectedErr:         newUnavailableError(context.DeadlineExceeded),
				},
			},
			expectedRunning: true,
		},
		{
			msg:                      "start then add",
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1"}},
				UpdateAction{AddedPeerIDs: []string{"2"}},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg:                      "start remove",
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"2"},
			releasedPeerIDs:          []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1", "2"}},
				UpdateAction{RemovedPeerIDs: []string{"1"}},
				ChooseAction{ExpectedPeer: "2"},
			},
			expectedRunning: true,
		},
		{
			msg:                      "start add many and remove many",
			retainedAvailablePeerIDs: []string{"1", "2", "3-r", "4-r", "5-a-r", "6-a-r", "7-a", "8-a"},
			releasedPeerIDs:          []string{"3-r", "4-r", "5-a-r", "6-a-r"},
			expectedAvailablePeers:   []string{"1", "2", "7-a", "8-a"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1", "2", "3-r", "4-r"}},
				UpdateAction{
					AddedPeerIDs: []string{"5-a-r", "6-a-r", "7-a", "8-a"},
				},
				UpdateAction{
					RemovedPeerIDs: []string{"5-a-r", "6-a-r", "3-r", "4-r"},
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "7-a"},
				ChooseAction{ExpectedPeer: "8-a"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg:                      "add duplicate peer",
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1", "2"}},
				UpdateAction{
					AddedPeerIDs: []string{"2"},
					ExpectedErr:  peer.ErrPeerAddAlreadyInList("2"),
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg:                      "remove peer not in list",
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1", "2"}},
				UpdateAction{
					RemovedPeerIDs: []string{"3"},
					ExpectedErr:    peer.ErrPeerRemoveNotInList("3"),
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg:                      "block but added too late",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				ConcurrentAction{
					Actions: []PeerListAction{
						ChooseAction{
							InputContextTimeout: 10 * time.Millisecond,
							ExpectedErr:         newUnavailableError(context.DeadlineExceeded),
						},
						UpdateAction{AddedPeerIDs: []string{"1"}},
					},
					Wait: 20 * time.Millisecond,
				},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg: "no blocking with no context deadline",
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{
					InputContext: context.Background(),
					ExpectedErr:  _noContextDeadlineError,
				},
			},
			expectedRunning: true,
		},
		{
			msg:                        "add unavailable peer",
			retainedAvailablePeerIDs:   []string{"1"},
			retainedUnavailablePeerIDs: []string{"2"},
			expectedAvailablePeers:     []string{"1"},
			expectedUnavailablePeers:   []string{"2"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1"}},
				UpdateAction{AddedPeerIDs: []string{"2"}},
				ChooseAction{
					ExpectedPeer:        "1",
					InputContextTimeout: 20 * time.Millisecond,
				},
				ChooseAction{
					ExpectedPeer:        "1",
					InputContextTimeout: 20 * time.Millisecond,
				},
			},
			expectedRunning: true,
		},
		{
			msg:                        "remove unavailable peer",
			retainedUnavailablePeerIDs: []string{"1"},
			releasedPeerIDs:            []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1"}},
				UpdateAction{RemovedPeerIDs: []string{"1"}},
				ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         newUnavailableError(context.DeadlineExceeded),
				},
			},
			expectedRunning: true,
		},
		{
			msg:                        "notify peer is now available",
			retainedUnavailablePeerIDs: []string{"1"},
			expectedAvailablePeers:     []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1"}},
				ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         newUnavailableError(context.DeadlineExceeded),
				},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Available},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg:                      "notify peer is still available",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1"}},
				ChooseAction{ExpectedPeer: "1"},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Available},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg:                      "notify peer is now unavailable",
			retainedAvailablePeerIDs: []string{"1"},
			expectedUnavailablePeers: []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1"}},
				ChooseAction{ExpectedPeer: "1"},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Unavailable},
				ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         newUnavailableError(context.DeadlineExceeded),
				},
			},
			expectedRunning: true,
		},
		{
			msg:                        "notify peer is still unavailable",
			retainedUnavailablePeerIDs: []string{"1"},
			expectedUnavailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1"}},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Unavailable},
				ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         newUnavailableError(context.DeadlineExceeded),
				},
			},
			expectedRunning: true,
		},
		{
			msg:                      "notify invalid peer",
			retainedAvailablePeerIDs: []string{"1"},
			releasedPeerIDs:          []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1"}},
				UpdateAction{RemovedPeerIDs: []string{"1"}},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Available},
			},
			expectedRunning: true,
		},
		{
			msg: "random insertion",
			// all scores are equal, degenerating to round-robin behavior
			// peer ordering is therefore by 'last'
			nextRand: nextRandFromSlice([]int{
				0, // insert p1 at end of list
				1, // insert p2 at end of list
				2, // insert p3 at end of list
				0, // swap p4 with index 0
				1, // swap p5 with index 1
			}),
			retainedAvailablePeerIDs: []string{"1", "2", "3", "4", "5"},
			expectedAvailablePeers:   []string{"1", "2", "3", "4", "5"},
			peerListActions: []PeerListAction{
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"1", "2", "3"}},
				UpdateAction{AddedPeerIDs: []string{"4", "5"}},
				ChooseAction{ExpectedPeer: "4"},
				ChooseAction{ExpectedPeer: "5"},
				ChooseAction{ExpectedPeer: "3"},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
			},
			expectedRunning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			transport := NewMockTransport(mockCtrl)

			// Healthy Transport Retain/Release
			peerMap := ExpectPeerRetains(
				transport,
				tt.retainedAvailablePeerIDs,
				tt.retainedUnavailablePeerIDs,
			)
			ExpectPeerReleases(transport, tt.releasedPeerIDs, nil)

			logger := zaptest.NewLogger(t)

			randOption := DisableRandomInsertion()
			if tt.nextRand != nil {
				randOption = InsertionOrder(tt.nextRand)
			}
			opts := []ListOption{Capacity(0), noShuffle, autoFlush, randOption, Logger(logger), Seed(0)}

			pl := New(transport, opts...)

			deps := ListActionDeps{
				Peers: peerMap,
			}
			ApplyPeerListActions(t, pl, tt.peerListActions, deps)

			var availablePeers []string
			var unavailablePeers []string
			for _, p := range pl.Peers() {
				ps := p.Status()
				if ps.ConnectionStatus == peer.Available {
					availablePeers = append(availablePeers, p.Identifier())
				} else if ps.ConnectionStatus == peer.Unavailable {
					unavailablePeers = append(unavailablePeers, p.Identifier())
				}
			}
			sort.Strings(availablePeers)
			sort.Strings(unavailablePeers)

			assert.Equal(t, tt.expectedAvailablePeers, availablePeers, "incorrect available peers")
			assert.Equal(t, tt.expectedUnavailablePeers, unavailablePeers, "incorrect unavailable peers")
			assert.Equal(t, tt.expectedRunning, pl.IsRunning(), "Peer list should match expected final running state")
		})
	}
}

var noShuffle ListOption = func(c *listConfig) {
	c.shuffle = false
}

var autoFlush ListOption = func(c *listConfig) {
	c.autoFlush = true
}

func TestFailFastConfig(t *testing.T) {
	conn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	require.NoError(t, conn.Close())

	serviceName := "test"
	config := whitespace.Expand(fmt.Sprintf(`
		outbounds:
			nowhere:
				http:
					fewest-pending-requests:
						peers:
							- %q
						capacity: 10
						failFast: true
	`, conn.Addr()))
	cfgr := yarpcconfig.New()
	cfgr.MustRegisterTransport(http.TransportSpec())
	cfgr.MustRegisterPeerList(Spec())
	cfg, err := cfgr.LoadConfigFromYAML(serviceName, strings.NewReader(config))
	require.NoError(t, err)

	d := yarpc.NewDispatcher(cfg)
	d.Start()
	defer d.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), testtime.Second)
	defer cancel()

	client := d.MustOutboundConfig("nowhere")
	_, err = client.Outbounds.Unary.Call(ctx, &transport.Request{
		Service:   "service",
		Caller:    "caller",
		Encoding:  transport.Encoding("blank"),
		Procedure: "bogus",
		Body:      strings.NewReader("nada"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no peer available")
}
