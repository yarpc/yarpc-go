package peerheap

import (
	"context"
	"sort"
	"testing"
	"time"

	"go.uber.org/yarpc/api/peer"
	. "go.uber.org/yarpc/api/peer/peertest"
	yerrors "go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/internal/sync"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPeerHeapList(t *testing.T) {
	type testStruct struct {
		msg string

		// PeerIDs that will be returned from the transport's OnRetain with "Available" status
		retainedAvailablePeerIDs []string

		// PeerIDs that will be returned from the transport's OnRetain with "Unavailable" status
		retainedUnavailablePeerIDs []string

		// PeerIDs that will be released from the transport
		releasedPeerIDs []string

		// PeerIDs that will return "retainErr" from the transport's OnRetain function
		errRetainedPeerIDs []string
		retainErr          error

		// PeerIDs that will return "releaseErr" from the transport's OnRelease function
		errReleasedPeerIDs []string
		releaseErr         error

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
			msg: "setup",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
			},
		},
		{
			msg: "setup with disconnected",
			retainedAvailablePeerIDs:   []string{"1"},
			retainedUnavailablePeerIDs: []string{"2"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1", "2"}},
			},
			expectedAvailablePeers:   []string{"1"},
			expectedUnavailablePeers: []string{"2"},
		},
		{
			msg: "start",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				ChooseAction{
					ExpectedPeer: "1",
				},
			},
			expectedRunning: true,
		},
		{
			msg: "start stop",
			retainedAvailablePeerIDs:   []string{"1", "2", "3", "4", "5", "6"},
			retainedUnavailablePeerIDs: []string{"7", "8", "9"},
			releasedPeerIDs:            []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}},
				StartAction{},
				StopAction{},
				ChooseAction{
					ExpectedErr:         sync.ErrAlreadyStopped,
					InputContextTimeout: 10 * time.Millisecond,
				},
			},
			expectedRunning: false,
		},
		{
			msg: "start many and choose",
			retainedAvailablePeerIDs: []string{"1", "2", "3", "4", "5", "6"},
			expectedAvailablePeers:   []string{"1", "2", "3", "4", "5", "6"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1", "2", "3", "4", "5", "6"}},
				StartAction{},
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
			msg: "assure start is idempotent",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				StartAction{},
				StartAction{},
				ChooseAction{
					ExpectedPeer: "1",
				},
			},
			expectedRunning: true,
		},
		{
			msg: "stop no start",
			retainedAvailablePeerIDs: []string{"1"},
			releasedPeerIDs:          []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StopAction{},
			},
			expectedRunning: false,
		},
		{
			msg:                "update retain error",
			errRetainedPeerIDs: []string{"1"},
			retainErr:          peer.ErrInvalidPeerType{},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}, ExpectedErr: peer.ErrInvalidPeerType{}},
			},
		},
		{
			msg: "update retain multiple errors",
			retainedAvailablePeerIDs: []string{"2"},
			errRetainedPeerIDs:       []string{"1", "3"},
			retainErr:                peer.ErrInvalidPeerType{},
			peerListActions: []PeerListAction{
				UpdateAction{
					AddedPeerIDs: []string{"1", "2", "3"},
					ExpectedErr:  yerrors.ErrorGroup{peer.ErrInvalidPeerType{}, peer.ErrInvalidPeerType{}},
				},
			},
			expectedAvailablePeers: []string{"2"},
		},
		{
			msg: "start stop release error",
			retainedAvailablePeerIDs: []string{"1"},
			errReleasedPeerIDs:       []string{"1"},
			releaseErr:               peer.ErrTransportHasNoReferenceToPeer{},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				StopAction{
					ExpectedErr: peer.ErrTransportHasNoReferenceToPeer{},
				},
			},
			expectedRunning: false,
		},
		{
			msg: "assure stop is idempotent",
			retainedAvailablePeerIDs: []string{"1"},
			errReleasedPeerIDs:       []string{"1"},
			releaseErr:               peer.ErrTransportHasNoReferenceToPeer{},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				ConcurrentAction{
					Actions: []PeerListAction{
						StopAction{
							ExpectedErr: peer.ErrTransportHasNoReferenceToPeer{},
						},
						StopAction{
							ExpectedErr: peer.ErrTransportHasNoReferenceToPeer{},
						},
						StopAction{
							ExpectedErr: peer.ErrTransportHasNoReferenceToPeer{},
						},
					},
				},
			},
			expectedRunning: false,
		},
		{
			msg: "start stop release multiple errors",
			retainedAvailablePeerIDs: []string{"1", "2", "3"},
			releasedPeerIDs:          []string{"2"},
			errReleasedPeerIDs:       []string{"1", "3"},
			releaseErr:               peer.ErrTransportHasNoReferenceToPeer{},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1", "2", "3"}},
				StartAction{},
				StopAction{
					ExpectedErr: yerrors.ErrorGroup{
						peer.ErrTransportHasNoReferenceToPeer{},
						peer.ErrTransportHasNoReferenceToPeer{},
					},
				},
			},
			expectedRunning: false,
		},
		{
			msg: "choose before start",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				ChooseAction{
					ExpectedErr:         context.DeadlineExceeded,
					InputContextTimeout: 10 * time.Millisecond,
				},
				ChooseAction{
					ExpectedErr:         context.DeadlineExceeded,
					InputContextTimeout: 10 * time.Millisecond,
				},
			},
			expectedRunning: false,
		},
		{
			msg: "start choose no peers",
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{
					InputContextTimeout: 20 * time.Millisecond,
					ExpectedErr:         context.DeadlineExceeded,
				},
			},
			expectedRunning: true,
		},
		{
			msg: "start then add",
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				UpdateAction{AddedPeerIDs: []string{"2"}},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg: "start remove",
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"2"},
			releasedPeerIDs:          []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1", "2"}},
				StartAction{},
				UpdateAction{RemovedPeerIDs: []string{"1"}},
				ChooseAction{ExpectedPeer: "2"},
			},
			expectedRunning: true,
		},
		{
			msg: "start add many and remove many",
			retainedAvailablePeerIDs: []string{"1", "2", "3-r", "4-r", "5-a-r", "6-a-r", "7-a", "8-a"},
			releasedPeerIDs:          []string{"3-r", "4-r", "5-a-r", "6-a-r"},
			expectedAvailablePeers:   []string{"1", "2", "7-a", "8-a"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1", "2", "3-r", "4-r"}},
				StartAction{},
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
			msg: "add retain error",
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			errRetainedPeerIDs:       []string{"3"},
			retainErr:                peer.ErrInvalidPeerType{},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1", "2"}},
				StartAction{},
				UpdateAction{
					AddedPeerIDs: []string{"3"},
					ExpectedErr:  peer.ErrInvalidPeerType{},
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg: "add duplicate peer",
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1", "2"}},
				StartAction{},
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
			msg: "remove peer not in list",
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1", "2"}},
				StartAction{},
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
			msg: "remove release error",
			retainedAvailablePeerIDs: []string{"1", "2"},
			errReleasedPeerIDs:       []string{"2"},
			releaseErr:               peer.ErrTransportHasNoReferenceToPeer{},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1", "2"}},
				StartAction{},
				UpdateAction{
					RemovedPeerIDs: []string{"2"},
					ExpectedErr:    peer.ErrTransportHasNoReferenceToPeer{},
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg: "block until add",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				ConcurrentAction{
					Actions: []PeerListAction{
						ChooseAction{
							InputContextTimeout: 200 * time.Millisecond,
							ExpectedPeer:        "1",
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
			msg: "multiple blocking until add",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				ConcurrentAction{
					Actions: []PeerListAction{
						ChooseAction{
							InputContextTimeout: 200 * time.Millisecond,
							ExpectedPeer:        "1",
						},
						ChooseAction{
							InputContextTimeout: 200 * time.Millisecond,
							ExpectedPeer:        "1",
						},
						ChooseAction{
							InputContextTimeout: 200 * time.Millisecond,
							ExpectedPeer:        "1",
						},
						UpdateAction{AddedPeerIDs: []string{"1"}},
					},
					Wait: 10 * time.Millisecond,
				},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg: "block but added too late",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				ConcurrentAction{
					Actions: []PeerListAction{
						ChooseAction{
							InputContextTimeout: 10 * time.Millisecond,
							ExpectedErr:         context.DeadlineExceeded,
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
			msg: "block until new peer after removal of only peer",
			retainedAvailablePeerIDs: []string{"1", "2"},
			releasedPeerIDs:          []string{"1"},
			expectedAvailablePeers:   []string{"2"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				UpdateAction{RemovedPeerIDs: []string{"1"}},
				ConcurrentAction{
					Actions: []PeerListAction{
						ChooseAction{
							InputContextTimeout: 200 * time.Millisecond,
							ExpectedPeer:        "2",
						},
						UpdateAction{AddedPeerIDs: []string{"2"}},
					},
					Wait: 20 * time.Millisecond,
				},
				ChooseAction{ExpectedPeer: "2"},
			},
			expectedRunning: true,
		},
		{
			msg: "no blocking with no context deadline",
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{
					InputContext: context.Background(),
					ExpectedErr:  peer.ErrChooseContextHasNoDeadline("PeerHeap"),
				},
			},
			expectedRunning: true,
		},
		{
			msg: "add unavailable peer",
			retainedAvailablePeerIDs:   []string{"1"},
			retainedUnavailablePeerIDs: []string{"2"},
			expectedAvailablePeers:     []string{"1"},
			expectedUnavailablePeers:   []string{"2"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
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
			msg: "remove unavailable peer",
			retainedUnavailablePeerIDs: []string{"1"},
			releasedPeerIDs:            []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				UpdateAction{RemovedPeerIDs: []string{"1"}},
				ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         context.DeadlineExceeded,
				},
			},
			expectedRunning: true,
		},
		{
			msg: "notify peer is now available",
			retainedUnavailablePeerIDs: []string{"1"},
			expectedAvailablePeers:     []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         context.DeadlineExceeded,
				},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Available},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg: "notify peer is still available",
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				ChooseAction{ExpectedPeer: "1"},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Available},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedRunning: true,
		},
		{
			msg: "notify peer is now unavailable",
			retainedAvailablePeerIDs: []string{"1"},
			expectedUnavailablePeers: []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				ChooseAction{ExpectedPeer: "1"},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Unavailable},
				ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         context.DeadlineExceeded,
				},
			},
			expectedRunning: true,
		},
		{
			msg: "notify peer is still unavailable",
			retainedUnavailablePeerIDs: []string{"1"},
			expectedUnavailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Unavailable},
				ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         context.DeadlineExceeded,
				},
			},
			expectedRunning: true,
		},
		{
			msg: "notify invalid peer",
			retainedAvailablePeerIDs: []string{"1"},
			releasedPeerIDs:          []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				UpdateAction{RemovedPeerIDs: []string{"1"}},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Available},
			},
			expectedRunning: true,
		},
		{
			// v: Available, u: Unavailable, a: Added, r: Removed
			msg: "notify peer stress test",
			retainedAvailablePeerIDs:   []string{"1v", "2va", "3vau", "4var", "5vaur"},
			retainedUnavailablePeerIDs: []string{"6u", "7ua", "8uav", "9uar", "10uavr"},
			releasedPeerIDs:            []string{"4var", "5vaur", "9uar", "10uavr"},
			expectedAvailablePeers:     []string{"1v", "2va", "8uav"},
			expectedUnavailablePeers:   []string{"3vau", "6u", "7ua"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1v", "6u"}},
				StartAction{},

				// Added Peers
				UpdateAction{
					AddedPeerIDs: []string{"2va", "3vau", "4var", "5vaur", "7ua", "8uav", "9uar", "10uavr"},
				},

				ChooseMultiAction{ExpectedPeers: []string{
					"1v", "2va", "3vau", "4var", "5vaur",
					"1v", "2va", "3vau", "4var", "5vaur",
				}},

				// Change Status to Unavailable
				NotifyStatusChangeAction{PeerID: "3vau", NewConnectionStatus: peer.Unavailable},
				NotifyStatusChangeAction{PeerID: "5vaur", NewConnectionStatus: peer.Unavailable},

				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "4var"}},
				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "4var"}},

				// Change Status to Available
				NotifyStatusChangeAction{PeerID: "8uav", NewConnectionStatus: peer.Available},
				NotifyStatusChangeAction{PeerID: "10uavr", NewConnectionStatus: peer.Available},

				ChooseMultiAction{ExpectedPeers: []string{"8uav", "10uavr"}}, // realign
				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "4var", "8uav", "10uavr"}},
				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "4var", "8uav", "10uavr"}},

				// Remove Peers
				UpdateAction{
					RemovedPeerIDs: []string{"4var", "5vaur", "9uar", "10uavr"},
				},

				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "8uav"}},
				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "8uav"}},
			},
			expectedRunning: true,
		},
		{
			msg: "block until notify available",
			retainedUnavailablePeerIDs: []string{"1"},
			expectedAvailablePeers:     []string{"1"},
			peerListActions: []PeerListAction{
				UpdateAction{AddedPeerIDs: []string{"1"}},
				StartAction{},
				ConcurrentAction{
					Actions: []PeerListAction{
						ChooseAction{
							InputContextTimeout: 200 * time.Millisecond,
							ExpectedPeer:        "1",
						},
						NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Available},
					},
					Wait: 20 * time.Millisecond,
				},
				ChooseAction{ExpectedPeer: "1"},
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

			// Unhealthy Transport Retain/Release
			ExpectPeerRetainsWithError(transport, tt.errRetainedPeerIDs, tt.retainErr)
			ExpectPeerReleases(transport, tt.errReleasedPeerIDs, tt.releaseErr)

			pl := New(transport)

			deps := ListActionDeps{
				Peers: peerMap,
			}
			ApplyPeerListActions(t, pl, tt.peerListActions, deps)

			var availablePeers []string
			var unavailablePeers []string
			for _, ps := range pl.byScore.peers {
				if ps.status.ConnectionStatus == peer.Available {
					availablePeers = append(availablePeers, ps.id.Identifier())
				} else if ps.status.ConnectionStatus == peer.Unavailable {
					unavailablePeers = append(unavailablePeers, ps.id.Identifier())
				}
			}
			sort.Strings(availablePeers)
			sort.Strings(unavailablePeers)

			assert.Equal(t, availablePeers, tt.expectedAvailablePeers, "incorrect available peers")
			assert.Equal(t, unavailablePeers, tt.expectedUnavailablePeers, "incorrect unavailable peers")
			assert.Equal(t, tt.expectedRunning, pl.IsRunning(), "Peer list should match expected final running state")
		})
	}
}
