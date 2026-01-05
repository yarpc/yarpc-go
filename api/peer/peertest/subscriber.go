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

package peertest

import (
	"github.com/golang/mock/gomock"
	"go.uber.org/yarpc/api/peer"
)

// SubscriberDefinition is an abstraction for defining a PeerSubscriber with
// an ID so it can be referenced later.
type SubscriberDefinition struct {
	ID                  string
	ExpectedNotifyCount int
}

// CreateSubscriberMap will take a slice of SubscriberDefinitions and return
// a map of IDs to MockPeerSubscribers
func CreateSubscriberMap(
	mockCtrl *gomock.Controller,
	subDefinitions []SubscriberDefinition,
) map[string]peer.Subscriber {
	subscribers := make(map[string]peer.Subscriber, len(subDefinitions))
	for _, subDef := range subDefinitions {
		sub := NewMockSubscriber(mockCtrl)
		sub.EXPECT().NotifyStatusChanged(gomock.Any()).Times(subDef.ExpectedNotifyCount)
		subscribers[subDef.ID] = sub
	}
	return subscribers
}
