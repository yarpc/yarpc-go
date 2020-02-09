// Copyright (c) 2020 Uber Technologies, Inc.
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

package yarpcpriority

import (
	"math/rand"
	"time"

	"go.uber.org/yarpc/api/priority"
)

// Option is an argument to the Prioritizer constructor.
type Option interface {
	apply(*Prioritizer)
}

// Options groups options.
func Options(opts ...Option) Option {
	return options(opts)
}

type options []Option

func (opts options) apply(p *Prioritizer) {
	for _, opt := range opts {
		opt.apply(p)
	}
}

// Priority overrides the priortizier's default priority of service.
func Priority(p priority.Priority) Option {
	return priorityOption{priority: p}
}

type priorityOption struct {
	priority priority.Priority
}

var _ Option = priorityOption{}

func (o priorityOption) apply(p *Prioritizer) {
	p.priority = o.priority
}

// ProcedurePriority overrides the default priority for requests for a
// particular procedure.
func ProcedurePriority(procedure string, priority priority.Priority) Option {
	return procedurePriorityOption{
		procedure: procedure,
		priority:  priority,
	}
}

type procedurePriorityOption struct {
	procedure string
	priority  priority.Priority
}

var _ Option = procedurePriorityOption{}

func (o procedurePriorityOption) apply(p *Prioritizer) {
	p.priorityRules[o.procedure] = o.priority
}

// FortuneRule is an option that adds a rule for generating a fortune for a
// request if it has either a header or trace header (baggage).
// The rule can be scoped to a specific procedure.
// By default, ever rule rotates the fortune every hour to prevent perpetual
// misfortune for the entity identified by a particular header.
// Permanent rules stick the priority for a header or baggage value
// permanently.
type FortuneRule struct {
	Header    string
	Baggage   string
	Procedure string
	Permanent bool
}

var _ Option = FortuneRule{}

func (r FortuneRule) apply(p *Prioritizer) {
	p.fortuneRules = append(p.fortuneRules, r)
}

// Random overrides the default source of random numbers.
func Random(source rand.Source) Option {
	return randomOption{source: source}
}

type randomOption struct {
	source rand.Source
}

var _ Option = randomOption{}

func (o randomOption) apply(p *Prioritizer) {
	p.prng = o.source
}

// Hash specifies a function from to generate a priority for a key.
//
// The default hash is CRC32.
func Hash(hash func(string) uint32) Option {
	return hashOption{hash: hash}
}

type hashOption struct {
	hash func(string) uint32
}

var _ Option = hashOption{}

func (o hashOption) apply(p *Prioritizer) {
	p.hash = o.hash
}

// TimeHash specifies a hash function from which to generate a priority for a
// key and time.
//
// The default hash incorporates CRC32 on the key and varies on the hour.
func TimeHash(timeHash func(string, time.Time) uint32) Option {
	return timeHashOption{timeHash: timeHash}
}

type timeHashOption struct {
	timeHash func(string, time.Time) uint32
}

var _ Option = timeHashOption{}

func (o timeHashOption) apply(p *Prioritizer) {
	p.timeHash = o.timeHash
}

// Time specifies an alternate time source.
func Time(now func() time.Time) Option {
	return timeOption{now: now}
}

type timeOption struct {
	now func() time.Time
}

var _ Option = timeOption{}

func (o timeOption) apply(p *Prioritizer) {
	p.now = o.now
}
