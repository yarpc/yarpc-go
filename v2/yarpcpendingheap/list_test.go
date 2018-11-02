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

package yarpcpendingheap

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcpeer"
	"go.uber.org/yarpc/v2/yarpctest"
)

var (
	_noContextDeadlineError = yarpcerror.Newf(yarpcerror.CodeInvalidArgument, "can't wait for peer without a context deadline for a fewest-pending-requests peer list")
)

func newNotRunningError(err string) error {
	return yarpcerror.FailedPreconditionErrorf("fewest-pending-requests peer list is not running: %s", err)
}

func newUnavailableError(err error) error {
	return yarpcerror.UnavailableErrorf("fewest-pending-requests peer list timed out waiting for peer: %s", err.Error())
}

func TestPeerHeapList(t *testing.T) {
	type testStruct struct {
		msg string

		// PeerIDs that will be returned from the dialer's OnRetain with "Available" status
		retainedAvailablePeerIDs []string

		// PeerIDs that will be returned from the dialer's OnRetain with "Unavailable" status
		retainedUnavailablePeerIDs []string

		// PeerIDs that will be released from the dialer
		releasedPeerIDs []string

		// PeerIDs that will return "retainErr" from the dialer's OnRetain function
		errRetainedPeerIDs []string
		retainErr          error

		// PeerIDs that will return "releaseErr" from the dialer's OnRelease function
		errReleasedPeerIDs []string
		releaseErr         error

		// A list of actions that will be applied on the PeerList
		peerListActions []yarpctest.PeerListAction

		// PeerIDs expected to be in the PeerList's "Available" list after the actions have been applied
		expectedAvailablePeers []string

		// PeerIDs expected to be in the PeerList's "Unavailable" list after the actions have been applied
		expectedUnavailablePeers []string
	}
	tests := []testStruct{
		{
			msg: "setup",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1"}},
			},
		},
		{
			msg: "setup with disconnected",
			retainedAvailablePeerIDs:   []string{"1"},
			retainedUnavailablePeerIDs: []string{"2"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1", "2"}},
			},
			expectedAvailablePeers:   []string{"1"},
			expectedUnavailablePeers: []string{"2"},
		},
		{
			msg: "update and choose",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1"}},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
			},
		},
		{
			msg: "update many many and choose",
			retainedAvailablePeerIDs: []string{"1", "2", "3", "4", "5", "6"},
			expectedAvailablePeers:   []string{"1", "2", "3", "4", "5", "6"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1", "2", "3", "4", "5", "6"}},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
				yarpctest.ChooseAction{ExpectedPeer: "2"},
				yarpctest.ChooseAction{ExpectedPeer: "3"},
				yarpctest.ChooseAction{ExpectedPeer: "4"},
				yarpctest.ChooseAction{ExpectedPeer: "5"},
				yarpctest.ChooseAction{ExpectedPeer: "6"},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
			},
		},
		{
			msg:                "update retain error",
			errRetainedPeerIDs: []string{"1"},
			retainErr:          yarpcpeer.ErrInvalidPeerType{},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1"}, ExpectedErr: yarpcpeer.ErrInvalidPeerType{}},
			},
		},
		{
			msg: "update retain multiple errors",
			retainedAvailablePeerIDs: []string{"2"},
			errRetainedPeerIDs:       []string{"1", "3"},
			retainErr:                yarpcpeer.ErrInvalidPeerType{},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{
					AddedPeerIDs: []string{"1", "2", "3"},
					ExpectedErr:  multierr.Combine(yarpcpeer.ErrInvalidPeerType{}, yarpcpeer.ErrInvalidPeerType{}),
				},
			},
			expectedAvailablePeers: []string{"2"},
		},
		{
			msg: "add retain error",
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			errRetainedPeerIDs:       []string{"3"},
			retainErr:                yarpcpeer.ErrInvalidPeerType{},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1", "2"}},
				yarpctest.UpdateAction{
					AddedPeerIDs: []string{"3"},
					ExpectedErr:  yarpcpeer.ErrInvalidPeerType{},
				},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
				yarpctest.ChooseAction{ExpectedPeer: "2"},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
			},
		},
		{
			msg: "add duplicate peer",
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1", "2"}},
				yarpctest.UpdateAction{
					AddedPeerIDs: []string{"2"},
					ExpectedErr:  yarpcpeer.ErrPeerAddAlreadyInList("2"),
				},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
				yarpctest.ChooseAction{ExpectedPeer: "2"},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
			},
		},
		{
			msg: "remove peer not in list",
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1", "2"}},
				yarpctest.UpdateAction{
					RemovedPeerIDs: []string{"3"},
					ExpectedErr:    yarpcpeer.ErrPeerRemoveNotInList("3"),
				},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
				yarpctest.ChooseAction{ExpectedPeer: "2"},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
			},
		},
		{
			msg: "remove release error",
			retainedAvailablePeerIDs: []string{"1", "2"},
			errReleasedPeerIDs:       []string{"2"},
			releaseErr:               yarpcpeer.ErrDialerHasNoReferenceToPeer{},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1", "2"}},
				yarpctest.UpdateAction{
					RemovedPeerIDs: []string{"2"},
					ExpectedErr:    yarpcpeer.ErrDialerHasNoReferenceToPeer{},
				},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
			},
		},
		{
			msg: "block but added too late",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.ConcurrentAction{
					Actions: []yarpctest.PeerListAction{
						yarpctest.ChooseAction{
							InputContextTimeout: 10 * time.Millisecond,
							ExpectedErr:         newUnavailableError(context.DeadlineExceeded),
						},
						yarpctest.UpdateAction{AddedPeerIDs: []string{"1"}},
					},
					Wait: 20 * time.Millisecond,
				},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
			},
		},
		{
			msg: "no blocking with no context deadline",
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.ChooseAction{
					InputContext: context.Background(),
					ExpectedErr:  _noContextDeadlineError,
				},
			},
		},
		{
			msg: "add unavailable peer",
			retainedAvailablePeerIDs:   []string{"1"},
			retainedUnavailablePeerIDs: []string{"2"},
			expectedAvailablePeers:     []string{"1"},
			expectedUnavailablePeers:   []string{"2"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1"}},
				yarpctest.UpdateAction{AddedPeerIDs: []string{"2"}},
				yarpctest.ChooseAction{
					ExpectedPeer:        "1",
					InputContextTimeout: 20 * time.Millisecond,
				},
				yarpctest.ChooseAction{
					ExpectedPeer:        "1",
					InputContextTimeout: 20 * time.Millisecond,
				},
			},
		},
		{
			msg: "remove unavailable peer",
			retainedUnavailablePeerIDs: []string{"1"},
			releasedPeerIDs:            []string{"1"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1"}},
				yarpctest.UpdateAction{RemovedPeerIDs: []string{"1"}},
				yarpctest.ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         newUnavailableError(context.DeadlineExceeded),
				},
			},
		},
		{
			msg: "notify peer is now available",
			retainedUnavailablePeerIDs: []string{"1"},
			expectedAvailablePeers:     []string{"1"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1"}},
				yarpctest.ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         newUnavailableError(context.DeadlineExceeded),
				},
				yarpctest.NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: yarpc.Available},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
			},
		},
		{
			msg: "notify peer is still available",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1"}},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
				yarpctest.NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: yarpc.Available},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
			},
		},
		{
			msg: "notify peer is now unavailable",
			retainedAvailablePeerIDs: []string{"1"},
			expectedUnavailablePeers: []string{"1"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1"}},
				yarpctest.ChooseAction{ExpectedPeer: "1"},
				yarpctest.NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: yarpc.Unavailable},
				yarpctest.ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         newUnavailableError(context.DeadlineExceeded),
				},
			},
		},
		{
			msg: "notify peer is still unavailable",
			retainedUnavailablePeerIDs: []string{"1"},
			expectedUnavailablePeers:   []string{"1"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1"}},
				yarpctest.NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: yarpc.Unavailable},
				yarpctest.ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         newUnavailableError(context.DeadlineExceeded),
				},
			},
		},
		{
			msg: "notify invalid peer",
			retainedAvailablePeerIDs: []string{"1"},
			releasedPeerIDs:          []string{"1"},
			peerListActions: []yarpctest.PeerListAction{
				yarpctest.UpdateAction{AddedPeerIDs: []string{"1"}},
				yarpctest.UpdateAction{RemovedPeerIDs: []string{"1"}},
				yarpctest.NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: yarpc.Available},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			dialer := yarpctest.NewMockDialer(mockCtrl)

			// Healthy Transport Retain/Release
			peerMap := yarpctest.ExpectPeerRetains(
				dialer,
				tt.retainedAvailablePeerIDs,
				tt.retainedUnavailablePeerIDs,
			)
			yarpctest.ExpectPeerReleases(dialer, tt.releasedPeerIDs, nil)

			// Unhealthy Transport Retain/Release
			yarpctest.ExpectPeerRetainsWithError(dialer, tt.errRetainedPeerIDs, tt.retainErr)
			yarpctest.ExpectPeerReleases(dialer, tt.errReleasedPeerIDs, tt.releaseErr)

			opts := []ListOption{Capacity(0), noShuffle}
			pl := New(dialer, opts...)

			deps := yarpctest.ListActionDeps{
				Peers: peerMap,
			}
			yarpctest.ApplyPeerListActions(t, pl, tt.peerListActions, deps)

			var availablePeers []string
			var unavailablePeers []string
			for _, p := range pl.Peers() {
				ps := p.Status()
				if ps.ConnectionStatus == yarpc.Available {
					availablePeers = append(availablePeers, p.Identifier())
				} else if ps.ConnectionStatus == yarpc.Unavailable {
					unavailablePeers = append(unavailablePeers, p.Identifier())
				}
			}
			sort.Strings(availablePeers)
			sort.Strings(unavailablePeers)

			assert.Equal(t, availablePeers, tt.expectedAvailablePeers, "incorrect available peers")
			assert.Equal(t, unavailablePeers, tt.expectedUnavailablePeers, "incorrect unavailable peers")
		})
	}
}

var noShuffle ListOption = func(c *listConfig) {
	c.shuffle = false
}
