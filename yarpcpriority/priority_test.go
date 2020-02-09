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
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/api/priority"
	"go.uber.org/yarpc/api/transport"
)

func TestRandomFortune(t *testing.T) {
	p := New(Time(func() time.Time { return time.Unix(0, 0) }))
	ctx := context.Background()
	var p1, p2 priority.Priority
	var f1, f2 priority.Priority

	{
		tracer := mocktracer.New()
		span := tracer.StartSpan("prioritized")
		ctx = opentracing.ContextWithSpan(ctx, span)
		p1, f1 = p.Priority(ctx, &transport.RequestMeta{})
	}

	{
		tracer := mocktracer.New()
		span := tracer.StartSpan("prioritized")
		ctx = opentracing.ContextWithSpan(ctx, span)
		p2, f2 = p.Priority(ctx, &transport.RequestMeta{})
	}

	assert.Equal(t, p1, p2)
	assert.NotEqual(t, f1, f2)
}

func TestPriorityDifferentiation(t *testing.T) {
	var p1, p2 priority.Priority

	{
		p := New(
			Priority(1),
			Time(func() time.Time { return time.Unix(0, 0) }),
		)
		ctx := context.Background()
		tracer := mocktracer.New()
		span := tracer.StartSpan("prioritized")
		ctx = opentracing.ContextWithSpan(ctx, span)
		p1, _ = p.Priority(ctx, &transport.RequestMeta{})
	}

	{
		p := New(
			Priority(2),
			Time(func() time.Time { return time.Unix(0, 0) }),
		)
		ctx := context.Background()
		tracer := mocktracer.New()
		span := tracer.StartSpan("prioritized")
		ctx = opentracing.ContextWithSpan(ctx, span)
		p2, _ = p.Priority(ctx, &transport.RequestMeta{})
	}

	assert.Less(t, uint8(p1), uint8(p2))
}

func TestConsistentHeaderFortune(t *testing.T) {
	p := New(
		FortuneRule{Header: "user"},
		Time(func() time.Time { return time.Unix(0, 0) }),
	)
	tracer := mocktracer.New()

	ctx := context.Background()
	span := tracer.StartSpan("prioritized")
	ctx = opentracing.ContextWithSpan(ctx, span)

	var p1, p2, p3 priority.Priority
	var f1, f2, f3 priority.Priority

	{
		p1, f1 = p.Priority(ctx, &transport.RequestMeta{
			Headers: transport.NewHeaders().With("user", "dk"),
		})
	}

	{
		p2, f2 = p.Priority(ctx, &transport.RequestMeta{
			Headers: transport.NewHeaders().With("user", "dk"),
		})

		// Verify that the priority was captured on the span.
		p3, f3 = p.Priority(ctx, &transport.RequestMeta{})
	}

	assert.Equal(t, p1, p2)
	assert.Equal(t, p1, p3)
	assert.Equal(t, f1, f2)
	assert.Equal(t, f1, f3)
	assert.NotEqual(t, priority.Lowest, f1)
}

func TestPermanentHeaderFortune(t *testing.T) {
	p := New(
		FortuneRule{Header: "user", Permanent: true},
	)
	tracer := mocktracer.New()

	ctx := context.Background()
	span := tracer.StartSpan("prioritized")
	ctx = opentracing.ContextWithSpan(ctx, span)

	var p1, p2, p3 priority.Priority
	var f1, f2, f3 priority.Priority

	{
		p1, f1 = p.Priority(ctx, &transport.RequestMeta{
			Headers: transport.NewHeaders().With("user", "tk"),
		})
	}

	{
		p2, f2 = p.Priority(ctx, &transport.RequestMeta{
			Headers: transport.NewHeaders().With("user", "tk"),
		})

		// Verify that the priority was captured on the span.
		p3, f3 = p.Priority(ctx, &transport.RequestMeta{})
	}

	assert.Equal(t, p1, p2)
	assert.Equal(t, p1, p3)
	assert.Equal(t, f1, f2)
	assert.Equal(t, f1, f3)
	assert.NotEqual(t, priority.Lowest, f1)
}

func TestProcedureFortune(t *testing.T) {
	p := New(
		Options(
			FortuneRule{Header: "user", Procedure: "foo", Permanent: true},
			FortuneRule{Header: "owner", Permanent: true},
		),
	)

	var p1, p2, p3 priority.Priority
	var f1, f2, f3 priority.Priority

	{
		tracer := mocktracer.New()
		ctx := context.Background()
		span := tracer.StartSpan("prioritized")
		ctx = opentracing.ContextWithSpan(ctx, span)
		p1, f1 = p.Priority(ctx, &transport.RequestMeta{
			Procedure: "foo",
			Headers:   transport.NewHeaders().With("user", "tk"),
		})
	}

	{
		tracer := mocktracer.New()
		ctx := context.Background()
		span := tracer.StartSpan("prioritized")
		ctx = opentracing.ContextWithSpan(ctx, span)
		p2, f2 = p.Priority(ctx, &transport.RequestMeta{
			Procedure: "bar",
			Headers:   transport.NewHeaders().With("owner", "tk"),
		})

		// Verify that the priority was captured on the span.
		p3, f3 = p.Priority(ctx, &transport.RequestMeta{})
	}

	assert.Equal(t, p1, p2)
	assert.Equal(t, p1, p3)
	assert.Equal(t, f1, f2)
	assert.Equal(t, f1, f3)
	assert.NotEqual(t, priority.Lowest, f1)
}

func TestProcedurePriority(t *testing.T) {
	p := New(
		ProcedurePriority("foo", 1),
		ProcedurePriority("bar", 2),
	)

	{
		ctx := context.Background()
		p, _ := p.Priority(ctx, &transport.RequestMeta{
			Procedure: "foo",
		})
		assert.Equal(t, uint(p), uint(1))
	}

	{
		ctx := context.Background()
		p, _ := p.Priority(ctx, &transport.RequestMeta{
			Procedure: "bar",
		})
		assert.Equal(t, uint(p), uint(2))
	}

}

func TestConsistentBaggageFortune(t *testing.T) {
	p := New(
		FortuneRule{Baggage: "user"},
		Time(func() time.Time { return time.Unix(0, 0) }),
	)
	ctx := context.Background()
	var p1, p2, p3 priority.Priority
	var f1, f2, f3 priority.Priority

	{
		tracer := mocktracer.New()
		span := tracer.StartSpan("prioritized")
		span.SetBaggageItem("user", "dk")
		ctx = opentracing.ContextWithSpan(ctx, span)
		p1, f1 = p.Priority(ctx, &transport.RequestMeta{})
	}

	{
		tracer := mocktracer.New()
		span := tracer.StartSpan("prioritized")
		span.SetBaggageItem("user", "dk")
		ctx = opentracing.ContextWithSpan(ctx, span)
		p2, f2 = p.Priority(ctx, &transport.RequestMeta{})

		// Verify that the priority was captured on the span.
		p3, f3 = p.Priority(ctx, &transport.RequestMeta{})
	}

	assert.Equal(t, p1, p2)
	assert.Equal(t, p1, p3)
	assert.Equal(t, f1, f2)
	assert.Equal(t, f1, f3)
	assert.NotEqual(t, priority.Lowest, f1)
}

func TestConsistentFortuneVaries(t *testing.T) {
	var now = time.Unix(0, 0)
	p := New(
		FortuneRule{Baggage: "user"},
		Time(func() time.Time { return now }),
	)
	ctx := context.Background()
	var p1, p2, p3 priority.Priority
	var f1, f2, f3 priority.Priority

	{
		tracer := mocktracer.New()
		span := tracer.StartSpan("prioritized")
		span.SetBaggageItem("user", "dk")
		ctx = opentracing.ContextWithSpan(ctx, span)
		p1, f1 = p.Priority(ctx, &transport.RequestMeta{})
	}

	// Almost one hour later.
	now = time.Unix(60*60-1, 0)

	{
		tracer := mocktracer.New()
		span := tracer.StartSpan("prioritized")
		span.SetBaggageItem("user", "dk")
		ctx = opentracing.ContextWithSpan(ctx, span)
		p2, f2 = p.Priority(ctx, &transport.RequestMeta{})
	}

	// A full hour later.
	now = time.Unix(60*60, 0)

	{
		tracer := mocktracer.New()
		span := tracer.StartSpan("prioritized")
		span.SetBaggageItem("user", "dk")
		ctx = opentracing.ContextWithSpan(ctx, span)
		p3, f3 = p.Priority(ctx, &transport.RequestMeta{})
	}

	assert.Equal(t, p1, p2)
	assert.Equal(t, p2, p3)

	assert.Equal(t, f1, f2)
	assert.NotEqual(t, f2, f3)
}

func TestExaggerationsAboutPriority(t *testing.T) {
	prioritizer := New()
	tracer := mocktracer.New()

	ctx := context.Background()
	span := tracer.StartSpan("prioritized")
	span.SetBaggageItem("priority", "200")
	span.SetBaggageItem("fortune", "200")
	ctx = opentracing.ContextWithSpan(ctx, span)
	p, f := prioritizer.Priority(ctx, &transport.RequestMeta{})

	assert.Equal(t, priority.Highest, p)
	assert.Equal(t, priority.Highest, f)
}

func TestLiesAboutPriority(t *testing.T) {
	prioritizer := New()
	tracer := mocktracer.New()

	ctx := context.Background()
	span := tracer.StartSpan("prioritized")
	span.SetBaggageItem("priority", "-200")
	span.SetBaggageItem("fortune", "-200")
	ctx = opentracing.ContextWithSpan(ctx, span)
	p, f := prioritizer.Priority(ctx, &transport.RequestMeta{})

	assert.Equal(t, priority.Lowest, p)
	assert.Equal(t, priority.Lowest, f)
}
