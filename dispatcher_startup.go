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

package yarpc

import (
	"errors"
	"sync"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errorsync"

	"go.uber.org/atomic"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// PhasedStarter is a more granular alternative to the Dispatcher's all-in-one
// Start method. Rather than starting the transports, inbounds, and outbounds
// in one call, it lets the user choose when to trigger each phase of
// dispatcher startup. For details on the interaction of Start and phased
// startup, see the documentation for the Dispatcher's PhasedStart method.
//
// The user of a PhasedStarter is responsible for correctly ordering startup:
// transports MUST be started before outbounds, which MUST be started before
// inbounds. Attempting startup in any other order will return an error.
type PhasedStarter struct {
	startedMu sync.Mutex
	started   []transport.Lifecycle

	dispatcher *Dispatcher
	log        *zap.Logger

	transportsStartInitiated atomic.Bool
	transportsStarted        atomic.Bool
	outboundsStartInitiated  atomic.Bool
	outboundsStarted         atomic.Bool
	inboundsStartInitiated   atomic.Bool
}

// StartTransports is the first step in startup. It starts all transports
// configured on the dispatcher, which is a necessary precondition for making
// and receiving RPCs. It's safe to call concurrently, but all calls after the
// first return an error.
func (s *PhasedStarter) StartTransports() error {
	if s.transportsStartInitiated.Swap(true) {
		return errors.New("already began starting transports")
	}
	defer s.transportsStarted.Store(true)
	s.log.Info("starting transports")
	wait := errorsync.ErrorWaiter{}
	for _, t := range s.dispatcher.transports {
		wait.Submit(s.start(t))
	}
	if errs := wait.Wait(); len(errs) != 0 {
		return s.abort(errs)
	}
	s.log.Debug("started transports")
	return nil
}

// StartOutbounds is the second phase of startup. It starts all outbounds
// configured on the dispatcher, which allows users of the dispatcher to
// construct clients and begin making outbound RPCs. It's safe to call
// concurrently, but all calls after the first return an error.
func (s *PhasedStarter) StartOutbounds() error {
	if !s.transportsStarted.Load() {
		return errors.New("must start outbounds after transports")
	}
	if s.outboundsStartInitiated.Swap(true) {
		return errors.New("already began starting outbounds")
	}
	defer s.outboundsStarted.Store(true)
	s.log.Info("starting outbounds")
	wait := errorsync.ErrorWaiter{}
	for _, o := range s.dispatcher.outbounds {
		wait.Submit(s.start(o.Unary))
		wait.Submit(s.start(o.Oneway))
		wait.Submit(s.start(o.Stream))
	}
	if errs := wait.Wait(); len(errs) != 0 {
		return s.abort(errs)
	}
	s.log.Debug("started outbounds")
	return nil
}

// StartInbounds is the final phase of startup. It starts all inbounds
// configured on the dispatcher, which allows any registered procedures to
// begin receiving requests. It's safe to call concurrently, but all calls
// after the first return an error.
func (s *PhasedStarter) StartInbounds() error {
	if !s.transportsStarted.Load() || !s.outboundsStarted.Load() {
		return errors.New("must start inbounds after transports and outbounds")
	}
	if s.inboundsStartInitiated.Swap(true) {
		return errors.New("already began starting inbounds")
	}
	s.log.Info("starting inbounds")
	wait := errorsync.ErrorWaiter{}
	for _, i := range s.dispatcher.inbounds {
		wait.Submit(s.start(i))
	}
	if errs := wait.Wait(); len(errs) != 0 {
		return s.abort(errs)
	}
	s.log.Debug("started inbounds")
	return nil
}

func (s *PhasedStarter) start(lc transport.Lifecycle) func() error {
	return func() error {
		if lc == nil {
			return nil
		}

		if err := lc.Start(); err != nil {
			return err
		}

		s.startedMu.Lock()
		s.started = append(s.started, lc)
		s.startedMu.Unlock()

		return nil
	}
}

func (s *PhasedStarter) abort(errs []error) error {
	// Failed to start so stop everything that was started.
	wait := errorsync.ErrorWaiter{}
	s.startedMu.Lock()
	for _, lc := range s.started {
		wait.Submit(lc.Stop)
	}
	s.startedMu.Unlock()
	if newErrors := wait.Wait(); len(newErrors) > 0 {
		errs = append(errs, newErrors...)
	}

	return multierr.Combine(errs...)
}

func (s *PhasedStarter) setRouters() {
	// Don't need synchronization, since we always call this in a lifecycle.Once
	// in the dispatcher.
	s.log.Debug("setting router for inbounds")
	for _, ib := range s.dispatcher.inbounds {
		ib.SetRouter(s.dispatcher.table)
	}
	s.log.Debug("set router for inbounds")
}

// PhasedStopper is a more granular alternative to the Dispatcher's all-in-one
// Stop method. Rather than stopping the inbounds, outbounds, and transports
// in one call, it lets the user choose when to trigger each phase of
// dispatcher shutdown. For details on the interaction of Stop and phased
// shutdown, see the documentation for the Dispatcher's PhasedStop method.
//
// The user of a PhasedStopper is responsible for correctly ordering shutdown:
// inbounds MUST be stopped before outbounds, which MUST be stopped before
// transports. Attempting shutdown in any other order will return an error.
type PhasedStopper struct {
	dispatcher *Dispatcher
	log        *zap.Logger

	inboundsStopInitiated   atomic.Bool
	inboundsStopped         atomic.Bool
	outboundsStopInitiated  atomic.Bool
	outboundsStopped        atomic.Bool
	transportsStopInitiated atomic.Bool
}

// StopInbounds is the first step in shutdown. It stops all inbounds
// configured on the dispatcher, which stops routing RPCs to all registered
// procedures. It's safe to call concurrently, but all calls after the first
// return an error.
func (s *PhasedStopper) StopInbounds() error {
	if s.inboundsStopInitiated.Swap(true) {
		return errors.New("already began stopping inbounds")
	}
	defer s.inboundsStopped.Store(true)
	s.log.Debug("stopping inbounds")
	wait := errorsync.ErrorWaiter{}
	for _, ib := range s.dispatcher.inbounds {
		wait.Submit(ib.Stop)
	}
	if errs := wait.Wait(); len(errs) > 0 {
		return multierr.Combine(errs...)
	}
	s.log.Debug("stopped inbounds")
	return nil
}

// StopOutbounds is the second step in shutdown. It stops all outbounds
// configured on the dispatcher, which stops clients from making outbound
// RPCs. It's safe to call concurrently, but all calls after the first return
// an error.
func (s *PhasedStopper) StopOutbounds() error {
	if !s.inboundsStopped.Load() {
		return errors.New("must stop inbounds first")
	}
	if s.outboundsStopInitiated.Swap(true) {
		return errors.New("already began stopping outbounds")
	}
	defer s.outboundsStopped.Store(true)
	s.log.Debug("stopping outbounds")
	wait := errorsync.ErrorWaiter{}
	for _, o := range s.dispatcher.outbounds {
		if o.Unary != nil {
			wait.Submit(o.Unary.Stop)
		}
		if o.Oneway != nil {
			wait.Submit(o.Oneway.Stop)
		}
		if o.Stream != nil {
			wait.Submit(o.Stream.Stop)
		}
	}
	if errs := wait.Wait(); len(errs) > 0 {
		return multierr.Combine(errs...)
	}
	s.log.Debug("stopped outbounds")
	return nil
}

// StopTransports is the final step in shutdown. It stops all transports
// configured on the dispatcher and cleans up any ancillary goroutines. It's
// safe to call concurrently, but all calls after the first return an error.
func (s *PhasedStopper) StopTransports() error {
	if !s.inboundsStopped.Load() || !s.outboundsStopped.Load() {
		return errors.New("must stop inbounds and outbounds first")
	}
	if s.transportsStopInitiated.Swap(true) {
		return errors.New("already began stopping transports")
	}
	s.log.Debug("stopping transports")
	wait := errorsync.ErrorWaiter{}
	for _, t := range s.dispatcher.transports {
		wait.Submit(t.Stop)
	}
	if errs := wait.Wait(); len(errs) > 0 {
		return multierr.Combine(errs...)
	}
	s.log.Debug("stopped transports")

	s.log.Debug("stopping metrics push loop, if any")
	s.dispatcher.stopMeter()
	s.log.Debug("stopped metrics push loop, if any")

	return nil
}
