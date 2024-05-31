// Copyright (c) 2024 Uber Technologies, Inc.
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

package peer

import (
	"context"

	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/api/x/introspection"
	"go.uber.org/yarpc/pkg/lifecycle"
)

// Bind couples a peer list with a peer list updater.
// Bind accepts a peer list and passes that peer list to a binder.
// The binder must return a peer list updater bound to the peer list.
// The peer list updater must implement Lifecycle so the bound peer list
// can start and stop updates.
func Bind(chooserList peer.ChooserList, bind peer.Binder) *BoundChooser {
	return &BoundChooser{
		once:        lifecycle.NewOnce(),
		chooserList: chooserList,
		updater:     bind(chooserList),
	}
}

// BoundChooser is a peer chooser that couples a peer list and a peer list
// updater for the duration of its lifecycle.
type BoundChooser struct {
	once        *lifecycle.Once
	updater     transport.Lifecycle // the peer list updater, which to us is just a lifecycle
	chooserList peer.ChooserList    // the peer list/chooser, also a lifecycle
}

// Updater returns the bound peer list updater.
func (c *BoundChooser) Updater() transport.Lifecycle {
	return c.updater
}

// ChooserList returns the bound peer list.
func (c *BoundChooser) ChooserList() peer.ChooserList {
	return c.chooserList
}

// Choose returns a peer from the bound peer list.
func (c *BoundChooser) Choose(ctx context.Context, treq *transport.Request) (peer peer.Peer, onFinish func(error), err error) {
	return c.chooserList.Choose(ctx, treq)
}

// Start starts the peer list and the peer list updater.
func (c *BoundChooser) Start() error {
	return c.once.Start(c.start)
}

func (c *BoundChooser) start() error {

	if err := c.chooserList.Start(); err != nil {
		return err
	}

	var errs error
	if err := c.updater.Start(); err != nil {
		errs = multierr.Append(errs, err)

		// Abort the peer chooser if the updater failed to start.
		if err := c.chooserList.Stop(); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	return errs
}

// Stop stops the peer list and the peer list updater.
func (c *BoundChooser) Stop() error {
	return c.once.Stop(c.stop)
}

func (c *BoundChooser) stop() error {
	var errs error

	if err := c.updater.Stop(); err != nil {
		errs = multierr.Append(errs, err)
	}

	if err := c.chooserList.Stop(); err != nil {
		errs = multierr.Append(errs, err)
	}

	return errs
}

// IsRunning returns whether the peer list and its peer list updater are both
// running, regardless of whether they should be running according to the bound
// chooser's lifecycle.
func (c *BoundChooser) IsRunning() bool {
	return c.chooserList.IsRunning() && c.updater.IsRunning()
}

// Introspect introspects the bound chooser.
func (c *BoundChooser) Introspect() introspection.ChooserStatus {
	if ic, ok := c.chooserList.(introspection.IntrospectableChooser); ok {
		return ic.Introspect()
	}
	return introspection.ChooserStatus{}
}

// BindPeers returns a binder (suitable as an argument to peer.Bind) that
// binds a peer list to a static list of peers for the duration of its
// lifecycle.
func BindPeers(ids []peer.Identifier) peer.Binder {
	return func(pl peer.List) transport.Lifecycle {
		return &PeersUpdater{
			once: lifecycle.NewOnce(),
			pl:   pl,
			ids:  ids,
		}
	}
}

// PeersUpdater binds a fixed list of peers to a peer list.
type PeersUpdater struct {
	once *lifecycle.Once
	pl   peer.List
	ids  []peer.Identifier
}

// Start adds a list of fixed peers to a peer list.
func (s *PeersUpdater) Start() error {
	return s.once.Start(s.start)
}

func (s *PeersUpdater) start() error {
	return s.pl.Update(peer.ListUpdates{
		Additions: s.ids,
	})
}

// Stop removes a list of fixed peers from a peer list.
func (s *PeersUpdater) Stop() error {
	return s.once.Stop(s.stop)
}

func (s *PeersUpdater) stop() error {
	return s.pl.Update(peer.ListUpdates{
		Removals: s.ids,
	})
}

// IsRunning returns whether the peers have been added and not removed.
func (s *PeersUpdater) IsRunning() bool {
	return s.once.IsRunning()
}
