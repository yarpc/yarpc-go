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
	"context"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"go.uber.org/yarpc/api/priority"
	"go.uber.org/yarpc/api/transport"
)

const (
	priorityBaggageKey = "priority"
	fortuneBaggageKey  = "fortune"
)

// Prioritizer extracts or creates priorities for requests.
type Prioritizer struct {
	mx sync.Mutex

	priority      priority.Priority
	priorityRules map[string]priority.Priority
	fortuneRules  []FortuneRule

	prng     rand.Source
	hash     func(string) uint32
	timeHash func(string, time.Time) uint32
	now      func() time.Time
}

var _ priority.Prioritizer = (*Prioritizer)(nil)

// New creates a prioritizer with the given options.
func New(opts ...Option) *Prioritizer {
	p := &Prioritizer{
		priorityRules: make(map[string]priority.Priority),
	}
	for _, opt := range opts {
		opt.apply(p)
	}
	if p.prng == nil {
		p.prng = rand.NewSource(time.Now().UnixNano())
	}
	if p.hash == nil {
		p.hash = makeDefaultHash()
	}
	if p.timeHash == nil {
		p.timeHash = makeDefaultTimeHash()
	}
	if p.now == nil {
		p.now = time.Now
	}
	return p
}

// Priority obtains or creates a priority and fortune for a context and
// request.
//
// The foremost preference is to use the "priority" and "fortune" headers
// present in context headers (trace baggge).
//
// In its absence, the prioritizer will create a priority and add it to the
// trace span.
//
// The default priority is 0.
// The default priority can be overridden with the Priority option or
// ProcedurePriority options if specific procedures have different default
// priorities.
//
// The default fortune is random.
// The default fortune can be overridden by the first FortuneRule option that
// matches the request procedure and a header or baggage.
// The priority will be consistent for the same header or baggage value for an
// hour at a time.
// Permanent fortune rules will consistently generate the same priority for the
// matched header or baggage regardless of the hour.
func (p *Prioritizer) Priority(ctx context.Context, req *transport.RequestMeta) (priority.Priority, priority.Priority) {
	var pn, fn priority.Priority
	var ps, fs string

	span := opentracing.SpanFromContext(ctx)

	if span != nil {
		ps = span.BaggageItem(priorityBaggageKey)
		fs = span.BaggageItem(fortuneBaggageKey)
	}

	if ps == "" {
		pn = p.constructPriority(req.Procedure)
		ps = strconv.Itoa(int(pn))
		if span != nil {
			span.SetBaggageItem(priorityBaggageKey, ps)
		}
	} else {
		if i, err := strconv.Atoi(ps); err == nil {
			pn = clamp(i)
		}
	}

	if fs == "" {
		fn = p.constructFortune(span, req)
		fs = strconv.Itoa(int(fn))
		if span != nil {
			span.SetBaggageItem(fortuneBaggageKey, fs)
		}
	} else {
		if i, err := strconv.Atoi(fs); err == nil {
			fn = clamp(i)
		}
	}

	return pn, fn
}

func clamp(i int) priority.Priority {
	if i > 100 {
		i = 100
	}
	if i < 0 {
		i = 0
	}
	return priority.Priority(i)
}

func (p *Prioritizer) constructPriority(procedure string) priority.Priority {
	if pn, ok := p.priorityRules[procedure]; ok {
		return pn
	}
	return p.priority
}

func (p *Prioritizer) constructFortune(span opentracing.Span, req *transport.RequestMeta) priority.Priority {
	for _, rule := range p.fortuneRules {
		var value string
		if rule.Procedure != "" && rule.Procedure != req.Procedure {
			continue
		}
		if rule.Baggage != "" {
			value = span.BaggageItem(rule.Baggage)
		}
		if rule.Header != "" {
			value, _ = req.Headers.Get(rule.Header)
		}
		if value != "" {
			var hash uint32
			if rule.Permanent {
				hash = p.lockHash(value)
			} else {
				hash = p.lockTimeHash(value)
			}
			return p.constructFortuneFromEntropy(hash)
		}
	}
	return p.constructRandomFortune()
}

func (p *Prioritizer) constructRandomFortune() priority.Priority {
	return p.constructFortuneFromEntropy(p.rand())
}

func (p *Prioritizer) constructFortuneFromEntropy(entropy uint32) priority.Priority {
	// Produce entropy in the range [1, 100] inclusive.
	return priority.Priority(entropy % 101)
}

func (p *Prioritizer) lockHash(key string) uint32 {
	p.mx.Lock()
	defer p.mx.Unlock()

	return p.hash(key)
}

func (p *Prioritizer) lockTimeHash(key string) uint32 {
	now := p.now()

	p.mx.Lock()
	defer p.mx.Unlock()

	return p.timeHash(key, now)
}

func (p *Prioritizer) rand() uint32 {
	p.mx.Lock()
	defer p.mx.Unlock()

	return uint32(p.prng.Int63())
}
