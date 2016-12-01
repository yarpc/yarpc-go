package roundrobin

import (
	"context"
	"fmt"
	"testing"
	"time"

	yerrors "go.uber.org/yarpc/internal/errors"
	"go.uber.org/yarpc/peer"
	. "go.uber.org/yarpc/peer/peertest"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRoundRobinList(t *testing.T) {
	type testStruct struct {
		msg string

		// PeerIDs that will be inserted into the PeerList at creation time
		inputPeerIDs []string

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

		// Expected Error to be returned from the PeerList's creation function
		expectedCreateErr error

		// PeerIDs expected to be in the PeerList's "Available" list after the actions have been applied
		expectedAvailablePeers []string

		// PeerIDs expected to be in the PeerList's "Unavailable" list after the actions have been applied
		expectedUnavailablePeers []string

		// Boolean indicating whether the PeerList is "started" after the actions have been applied
		expectedStarted bool
	}
	tests := []testStruct{
		{
			msg:                      "setup",
			inputPeerIDs:             []string{"1"},
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
		},
		{
			msg:                        "setup with disconnected",
			inputPeerIDs:               []string{"1", "2"},
			retainedAvailablePeerIDs:   []string{"1"},
			retainedUnavailablePeerIDs: []string{"2"},
			expectedAvailablePeers:     []string{"1"},
			expectedUnavailablePeers:   []string{"2"},
		},
		{
			msg:                      "start",
			inputPeerIDs:             []string{"1"},
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{
					ExpectedPeer: "1",
				},
			},
			expectedStarted: true,
		},
		{
			msg:                        "start stop",
			inputPeerIDs:               []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
			retainedAvailablePeerIDs:   []string{"1", "2", "3", "4", "5", "6"},
			retainedUnavailablePeerIDs: []string{"7", "8", "9"},
			releasedPeerIDs:            []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
			peerListActions: []PeerListAction{
				StartAction{},
				StopAction{},
				ChooseAction{
					ExpectedErr: peer.ErrPeerListNotStarted("RoundRobinList"),
				},
			},
			expectedStarted: false,
		},
		{
			msg:                      "start many and choose",
			inputPeerIDs:             []string{"1", "2", "3", "4", "5", "6"},
			retainedAvailablePeerIDs: []string{"1", "2", "3", "4", "5", "6"},
			expectedAvailablePeers:   []string{"1", "2", "3", "4", "5", "6"},
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "3"},
				ChooseAction{ExpectedPeer: "4"},
				ChooseAction{ExpectedPeer: "5"},
				ChooseAction{ExpectedPeer: "6"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:                      "start twice",
			inputPeerIDs:             []string{"1"},
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				StartAction{
					ExpectedErr: peer.ErrPeerListAlreadyStarted("RoundRobinList"),
				},
				ChooseAction{
					ExpectedPeer: "1",
				},
			},
			expectedStarted: true,
		},
		{
			msg:                      "stop no start",
			inputPeerIDs:             []string{"1"},
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StopAction{
					ExpectedErr: peer.ErrPeerListNotStarted("RoundRobinList"),
				},
			},
			expectedStarted: false,
		},
		{
			msg:                "start retain error",
			inputPeerIDs:       []string{"1"},
			errRetainedPeerIDs: []string{"1"},
			retainErr:          peer.ErrInvalidPeerType{},
			expectedCreateErr:  peer.ErrInvalidPeerType{},
		},
		{
			msg:                      "start retain multiple errors",
			inputPeerIDs:             []string{"1", "2", "3"},
			retainedAvailablePeerIDs: []string{"2"},
			errRetainedPeerIDs:       []string{"1", "3"},
			retainErr:                peer.ErrInvalidPeerType{},
			expectedCreateErr:        yerrors.ErrorGroup{peer.ErrInvalidPeerType{}, peer.ErrInvalidPeerType{}},
			expectedAvailablePeers:   []string{"2"},
		},
		{
			msg:                      "start stop release error",
			inputPeerIDs:             []string{"1"},
			retainedAvailablePeerIDs: []string{"1"},
			errReleasedPeerIDs:       []string{"1"},
			releaseErr:               peer.ErrTransportHasNoReferenceToPeer{},
			peerListActions: []PeerListAction{
				StartAction{},
				StopAction{
					ExpectedErr: peer.ErrTransportHasNoReferenceToPeer{},
				},
			},
			expectedStarted: false,
		},
		{
			msg:                      "start stop release multiple errors",
			inputPeerIDs:             []string{"1", "2", "3"},
			retainedAvailablePeerIDs: []string{"1", "2", "3"},
			releasedPeerIDs:          []string{"2"},
			errReleasedPeerIDs:       []string{"1", "3"},
			releaseErr:               peer.ErrTransportHasNoReferenceToPeer{},
			peerListActions: []PeerListAction{
				StartAction{},
				StopAction{
					ExpectedErr: yerrors.ErrorGroup{
						peer.ErrTransportHasNoReferenceToPeer{},
						peer.ErrTransportHasNoReferenceToPeer{},
					},
				},
			},
			expectedStarted: false,
		},
		{
			msg:                      "choose before start",
			inputPeerIDs:             []string{"1"},
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				ChooseAction{
					ExpectedErr: peer.ErrPeerListNotStarted("RoundRobinList"),
				},
				ChooseAction{
					ExpectedErr: peer.ErrPeerListNotStarted("RoundRobinList"),
				},
			},
			expectedStarted: false,
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
			expectedStarted: true,
		},
		{
			msg:                      "start add",
			inputPeerIDs:             []string{"1"},
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			peerListActions: []PeerListAction{
				StartAction{},
				AddAction{InputPeerID: "2"},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:                      "start remove",
			inputPeerIDs:             []string{"1", "2"},
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"2"},
			releasedPeerIDs:          []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				RemoveAction{InputPeerID: "1"},
				ChooseAction{ExpectedPeer: "2"},
			},
			expectedStarted: true,
		},
		{
			msg:                      "start add many and remove many",
			inputPeerIDs:             []string{"1", "2", "3-r", "4-r"},
			retainedAvailablePeerIDs: []string{"1", "2", "3-r", "4-r", "5-a-r", "6-a-r", "7-a", "8-a"},
			releasedPeerIDs:          []string{"3-r", "4-r", "5-a-r", "6-a-r"},
			expectedAvailablePeers:   []string{"1", "2", "7-a", "8-a"},
			peerListActions: []PeerListAction{
				StartAction{},
				AddAction{InputPeerID: "5-a-r"},
				AddAction{InputPeerID: "6-a-r"},
				AddAction{InputPeerID: "7-a"},
				AddAction{InputPeerID: "8-a"},
				RemoveAction{InputPeerID: "5-a-r"},
				RemoveAction{InputPeerID: "6-a-r"},
				RemoveAction{InputPeerID: "3-r"},
				RemoveAction{InputPeerID: "4-r"},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "7-a"},
				ChooseAction{ExpectedPeer: "8-a"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:                      "add retain error",
			inputPeerIDs:             []string{"1", "2"},
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			errRetainedPeerIDs:       []string{"3"},
			retainErr:                peer.ErrInvalidPeerType{},
			peerListActions: []PeerListAction{
				StartAction{},
				AddAction{
					InputPeerID: "3",
					ExpectedErr: peer.ErrInvalidPeerType{},
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:                      "add duplicate peer",
			inputPeerIDs:             []string{"1", "2"},
			retainedAvailablePeerIDs: []string{"1", "2", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			peerListActions: []PeerListAction{
				StartAction{},
				AddAction{
					InputPeerID: "2",
					ExpectedErr: peer.ErrPeerAddAlreadyInList("2"),
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:                      "remove peer not in list",
			inputPeerIDs:             []string{"1", "2"},
			retainedAvailablePeerIDs: []string{"1", "2"},
			expectedAvailablePeers:   []string{"1", "2"},
			peerListActions: []PeerListAction{
				StartAction{},
				RemoveAction{
					InputPeerID: "3",
					ExpectedErr: peer.ErrPeerRemoveNotInList("3"),
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "2"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:                      "remove release error",
			inputPeerIDs:             []string{"1", "2"},
			retainedAvailablePeerIDs: []string{"1", "2"},
			errReleasedPeerIDs:       []string{"2"},
			releaseErr:               peer.ErrTransportHasNoReferenceToPeer{},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				RemoveAction{
					InputPeerID: "2",
					ExpectedErr: peer.ErrTransportHasNoReferenceToPeer{},
				},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
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
						AddAction{InputPeerID: "1"},
					},
					Wait: 20 * time.Millisecond,
				},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
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
						AddAction{InputPeerID: "1"},
					},
					Wait: 10 * time.Millisecond,
				},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
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
						AddAction{InputPeerID: "1"},
					},
					Wait: 20 * time.Millisecond,
				},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:                      "block until new peer after removal of only peer",
			inputPeerIDs:             []string{"1"},
			retainedAvailablePeerIDs: []string{"1", "2"},
			releasedPeerIDs:          []string{"1"},
			expectedAvailablePeers:   []string{"2"},
			peerListActions: []PeerListAction{
				StartAction{},
				RemoveAction{InputPeerID: "1"},
				ConcurrentAction{
					Actions: []PeerListAction{
						ChooseAction{
							InputContextTimeout: 200 * time.Millisecond,
							ExpectedPeer:        "2",
						},
						AddAction{InputPeerID: "2"},
					},
					Wait: 20 * time.Millisecond,
				},
				ChooseAction{ExpectedPeer: "2"},
			},
			expectedStarted: true,
		},
		{
			msg: "no blocking with no context deadline",
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{
					InputContext: context.Background(),
					ExpectedErr:  peer.ErrChooseContextHasNoDeadline("RoundRobinList"),
				},
			},
			expectedStarted: true,
		},
		{
			msg:                        "add unavailable peer",
			inputPeerIDs:               []string{"1"},
			retainedAvailablePeerIDs:   []string{"1"},
			retainedUnavailablePeerIDs: []string{"2"},
			expectedAvailablePeers:     []string{"1"},
			expectedUnavailablePeers:   []string{"2"},
			peerListActions: []PeerListAction{
				StartAction{},
				AddAction{InputPeerID: "2"},
				ChooseAction{ExpectedPeer: "1"},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:                        "remove unavailable peer",
			inputPeerIDs:               []string{"1"},
			retainedUnavailablePeerIDs: []string{"1"},
			releasedPeerIDs:            []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				RemoveAction{InputPeerID: "1"},
				ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         context.DeadlineExceeded,
				},
			},
			expectedStarted: true,
		},
		{
			msg:                        "notify peer is now available",
			inputPeerIDs:               []string{"1"},
			retainedUnavailablePeerIDs: []string{"1"},
			expectedAvailablePeers:     []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         context.DeadlineExceeded,
				},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Available},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:                      "notify peer is still available",
			inputPeerIDs:             []string{"1"},
			retainedAvailablePeerIDs: []string{"1"},
			expectedAvailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{ExpectedPeer: "1"},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Available},
				ChooseAction{ExpectedPeer: "1"},
			},
			expectedStarted: true,
		},
		{
			msg:                      "notify peer is now unavailable",
			inputPeerIDs:             []string{"1"},
			retainedAvailablePeerIDs: []string{"1"},
			expectedUnavailablePeers: []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				ChooseAction{ExpectedPeer: "1"},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Unavailable},
				ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         context.DeadlineExceeded,
				},
			},
			expectedStarted: true,
		},
		{
			msg:                        "notify peer is still unavailable",
			inputPeerIDs:               []string{"1"},
			retainedUnavailablePeerIDs: []string{"1"},
			expectedUnavailablePeers:   []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Unavailable},
				ChooseAction{
					InputContextTimeout: 10 * time.Millisecond,
					ExpectedErr:         context.DeadlineExceeded,
				},
			},
			expectedStarted: true,
		},
		{
			msg:                      "notify invalid peer",
			inputPeerIDs:             []string{"1"},
			retainedAvailablePeerIDs: []string{"1"},
			releasedPeerIDs:          []string{"1"},
			peerListActions: []PeerListAction{
				StartAction{},
				RemoveAction{InputPeerID: "1"},
				NotifyStatusChangeAction{PeerID: "1", NewConnectionStatus: peer.Available},
			},
			expectedStarted: true,
		},
		{
			// v: Available, u: Unavailable, a: Added, r: Removed
			msg:                        "notify peer stress test",
			inputPeerIDs:               []string{"1v", "6u"},
			retainedAvailablePeerIDs:   []string{"1v", "2va", "3vau", "4var", "5vaur"},
			retainedUnavailablePeerIDs: []string{"6u", "7ua", "8uav", "9uar", "10uavr"},
			releasedPeerIDs:            []string{"4var", "5vaur", "9uar", "10uavr"},
			expectedAvailablePeers:     []string{"1v", "2va", "8uav"},
			expectedUnavailablePeers:   []string{"3vau", "6u", "7ua"},
			peerListActions: []PeerListAction{
				StartAction{},
				// Added Peers
				AddAction{InputPeerID: "2va"},
				AddAction{InputPeerID: "3vau"},
				AddAction{InputPeerID: "4var"},
				AddAction{InputPeerID: "5vaur"},
				AddAction{InputPeerID: "7ua"},
				AddAction{InputPeerID: "8uav"},
				AddAction{InputPeerID: "9uar"},
				AddAction{InputPeerID: "10uavr"},

				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "3vau", "4var", "5vaur"}},
				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "3vau", "4var", "5vaur"}},

				// Change Status to Unavailable
				NotifyStatusChangeAction{PeerID: "3vau", NewConnectionStatus: peer.Unavailable},
				NotifyStatusChangeAction{PeerID: "5vaur", NewConnectionStatus: peer.Unavailable},

				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "4var"}},
				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "4var"}},

				// Change Status to Available
				NotifyStatusChangeAction{PeerID: "8uav", NewConnectionStatus: peer.Available},
				NotifyStatusChangeAction{PeerID: "10uavr", NewConnectionStatus: peer.Available},

				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "4var", "8uav", "10uavr"}},
				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "4var", "8uav", "10uavr"}},

				// Remove Peers
				RemoveAction{InputPeerID: "4var"},
				RemoveAction{InputPeerID: "5vaur"},
				RemoveAction{InputPeerID: "9uar"},
				RemoveAction{InputPeerID: "10uavr"},

				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "8uav"}},
				ChooseMultiAction{ExpectedPeers: []string{"1v", "2va", "8uav"}},
			},
			expectedStarted: true,
		},
		{
			msg:                        "block until notify available",
			inputPeerIDs:               []string{"1"},
			retainedUnavailablePeerIDs: []string{"1"},
			expectedAvailablePeers:     []string{"1"},
			peerListActions: []PeerListAction{
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
			expectedStarted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			pids := CreatePeerIDs(tt.inputPeerIDs)
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

			pl, err := New(pids, transport)
			assert.Equal(t, tt.expectedCreateErr, err)

			deps := ListActionDeps{
				Peers: peerMap,
			}
			ApplyPeerListActions(t, pl, tt.peerListActions, deps)

			assert.Len(t, pl.availablePeerRing.peerToNode, len(tt.expectedAvailablePeers), "invalid available peerlist size")
			for _, expectedRingPeer := range tt.expectedAvailablePeers {
				node, ok := pl.availablePeerRing.peerToNode[expectedRingPeer]
				assert.True(t, ok, fmt.Sprintf("expected peer: %s was not in available peerlist", expectedRingPeer))
				if ok {
					actualPeer := getPeerForRingNode(node)
					assert.Equal(t, expectedRingPeer, actualPeer.Identifier())
				}
			}

			assert.Len(t, pl.unavailablePeers, len(tt.expectedUnavailablePeers), "invalid unavailable peerlist size")
			for _, expectedUnavailablePeer := range tt.expectedUnavailablePeers {
				p, ok := pl.unavailablePeers[expectedUnavailablePeer]
				assert.True(t, ok, fmt.Sprintf("expected peer: %s was not in unavailable peerlist", expectedUnavailablePeer))
				if ok {
					assert.Equal(t, expectedUnavailablePeer, p.Identifier())
				}
			}

			assert.Equal(t, tt.expectedStarted, pl.started.Load())
		})
	}
}
