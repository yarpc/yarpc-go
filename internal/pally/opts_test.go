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

package pally

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptsValidation(t *testing.T) {
	tests := []struct {
		desc  string
		opts  Opts
		ok    bool
		vecOK bool
	}{
		{
			desc: "valid names",
			opts: Opts{
				Name: "fOo123",
				Help: "Some help.",
			},
			ok:    true,
			vecOK: false,
		},
		{
			desc: "valid names & constant labels",
			opts: Opts{
				Name:        "foo",
				Help:        "Some help.",
				ConstLabels: Labels{"foo": "bar"},
			},
			ok:    true,
			vecOK: false,
		},
		{
			desc: "name with Tally-forbidden characters",
			opts: Opts{
				Name: "foo:bar",
				Help: "Some help.",
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "no name",
			opts: Opts{
				Help: "Some help.",
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "no help",
			opts: Opts{
				Name: "foo",
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "valid names but invalid label key",
			opts: Opts{
				Name:        "foo",
				Help:        "Some help.",
				ConstLabels: Labels{"foo:foo": "bar"},
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "valid names but invalid label value",
			opts: Opts{
				Name:        "foo",
				Help:        "Some help.",
				ConstLabels: Labels{"foo": "bar:bar"},
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "valid names & variable labels",
			opts: Opts{
				Name:           "foo",
				Help:           "Some help.",
				VariableLabels: []string{"baz"},
			},
			ok:    true,
			vecOK: true,
		},
		{
			desc: "valid names, constant labels, & variable labels",
			opts: Opts{
				Name:           "foo",
				Help:           "Some help.",
				ConstLabels:    Labels{"foo": "bar"},
				VariableLabels: []string{"baz"},
			},
			ok:    true,
			vecOK: true,
		},
		{
			desc: "valid names & constant labels, but invalid variable labels",
			opts: Opts{
				Name:           "foo",
				Help:           "Some help.",
				ConstLabels:    Labels{"foo": "bar"},
				VariableLabels: []string{"baz:baz"},
			},
			ok:    false, // Prometheus always validates the VariableLabels.
			vecOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.ok {
				assertSimpleOptsOK(t, tt.opts)
			} else {
				assertSimpleOptsFail(t, tt.opts)
			}
			if tt.vecOK {
				assertVectorOptsOK(t, tt.opts)
			} else {
				assertVectorOptsFail(t, tt.opts)
			}
		})
	}
}

func TestLatencyOptsValidation(t *testing.T) {
	tests := []struct {
		desc  string
		opts  LatencyOpts
		ok    bool
		vecOK bool
	}{
		{
			desc: "valid names",
			opts: LatencyOpts{
				Opts: Opts{
					Name: "fOo123",
					Help: "Some help.",
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			ok:    true,
			vecOK: false,
		},
		{
			desc: "valid names & constant labels",
			opts: LatencyOpts{
				Opts: Opts{
					Name:        "foo",
					Help:        "Some help.",
					ConstLabels: Labels{"foo": "bar"},
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			ok:    true,
			vecOK: false,
		},
		{
			desc: "name with Tally-forbidden characters",
			opts: LatencyOpts{
				Opts: Opts{
					Name: "foo:bar",
					Help: "Some help.",
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "no name",
			opts: LatencyOpts{
				Opts: Opts{
					Help: "Some help.",
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "no help",
			opts: LatencyOpts{
				Opts: Opts{
					Name: "foo",
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "valid names but invalid label key",
			opts: LatencyOpts{
				Opts: Opts{
					Name:        "foo",
					Help:        "Some help.",
					ConstLabels: Labels{"foo:foo": "bar"},
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "valid names but invalid label value",
			opts: LatencyOpts{
				Opts: Opts{
					Name:        "foo",
					Help:        "Some help.",
					ConstLabels: Labels{"foo": "bar:bar"},
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "valid names & variable labels",
			opts: LatencyOpts{
				Opts: Opts{
					Name:           "foo",
					Help:           "Some help.",
					VariableLabels: []string{"baz"},
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			ok:    true,
			vecOK: true,
		},
		{
			desc: "valid names, constant labels, & variable labels",
			opts: LatencyOpts{
				Opts: Opts{
					Name:           "foo",
					Help:           "Some help.",
					ConstLabels:    Labels{"foo": "bar"},
					VariableLabels: []string{"baz"},
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			ok:    true,
			vecOK: true,
		},
		{
			desc: "valid names & constant labels, but invalid variable labels",
			opts: LatencyOpts{
				Opts: Opts{
					Name:           "foo",
					Help:           "Some help.",
					ConstLabels:    Labels{"foo": "bar"},
					VariableLabels: []string{"baz:baz"},
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			ok:    false, // Prometheus always validates the VariableLabels.
			vecOK: false,
		},
		{
			desc: "valid labels, no unit",
			opts: LatencyOpts{
				Opts: Opts{
					Name:           "foo",
					Help:           "Some help.",
					ConstLabels:    Labels{"foo": "bar"},
					VariableLabels: []string{"baz"},
				},
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "valid labels, negative unit",
			opts: LatencyOpts{
				Opts: Opts{
					Name:           "foo",
					Help:           "Some help.",
					ConstLabels:    Labels{"foo": "bar"},
					VariableLabels: []string{"baz"},
				},
				Unit:    -1 * time.Millisecond,
				Buckets: []time.Duration{time.Second, time.Minute},
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "valid labels, no buckets",
			opts: LatencyOpts{
				Opts: Opts{
					Name:           "foo",
					Help:           "Some help.",
					ConstLabels:    Labels{"foo": "bar"},
					VariableLabels: []string{"baz"},
				},
				Unit: time.Millisecond,
			},
			ok:    false,
			vecOK: false,
		},
		{
			desc: "valid labels, buckets out of order",
			opts: LatencyOpts{
				Opts: Opts{
					Name:           "foo",
					Help:           "Some help.",
					ConstLabels:    Labels{"foo": "bar"},
					VariableLabels: []string{"baz"},
				},
				Unit:    time.Millisecond,
				Buckets: []time.Duration{time.Minute, time.Second},
			},
			ok:    false,
			vecOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.ok {
				assertSimpleLatencyOptsOK(t, tt.opts)
			} else {
				assertSimpleLatencyOptsFail(t, tt.opts)
			}
			if tt.vecOK {
				assertVectorLatencyOptsOK(t, tt.opts)
			} else {
				assertVectorLatencyOptsFail(t, tt.opts)
			}
		})
	}
}

func assertSimpleOptsOK(t testing.TB, opts Opts) {
	_, err := NewRegistry().NewCounter(opts)
	assert.NoError(t, err, "Expected success from NewCounter.")
	assert.NotPanics(t, func() { NewRegistry().MustCounter(opts) }, "Expected MustCounter to succeed.")

	_, err = NewRegistry().NewGauge(opts)
	assert.NoError(t, err, "Expected success from NewGauge.")
	assert.NotPanics(t, func() { NewRegistry().MustGauge(opts) }, "Expected a panic from MustGauge.")
}

func assertSimpleOptsFail(t testing.TB, opts Opts) {
	_, err := NewRegistry().NewCounter(opts)
	assert.Error(t, err, "Expected an error from NewCounter.")
	assert.Panics(t, func() { NewRegistry().MustCounter(opts) }, "Expected a panic from MustCounter.")

	_, err = NewRegistry().NewGauge(opts)
	assert.Error(t, err, "Expected an error from NewGauge.")
	assert.Panics(t, func() { NewRegistry().MustGauge(opts) }, "Expected a panic from MustGauge.")
}

func assertVectorOptsOK(t testing.TB, opts Opts) {
	_, err := NewRegistry().NewCounterVector(opts)
	assert.NoError(t, err, "Expected success from NewCounterVector.")
	assert.NotPanics(t, func() { NewRegistry().MustCounterVector(opts) }, "Expected MustCounterVector to succeed.")

	_, err = NewRegistry().NewGaugeVector(opts)
	assert.NoError(t, err, "Expected success from NewGaugeVector.")
	assert.NotPanics(t, func() { NewRegistry().MustGaugeVector(opts) }, "Expected a panic from MustGaugeVector.")
}

func assertVectorOptsFail(t testing.TB, opts Opts) {
	_, err := NewRegistry().NewCounterVector(opts)
	assert.Error(t, err, "Expected an error from NewCounterVector.")
	assert.Panics(t, func() { NewRegistry().MustCounterVector(opts) }, "Expected a panic from MustCounterVector.")

	_, err = NewRegistry().NewGaugeVector(opts)
	assert.Error(t, err, "Expected an error from NewGaugeVector.")
	assert.Panics(t, func() { NewRegistry().MustGaugeVector(opts) }, "Expected a panic from MustGaugeVector.")
}

func assertSimpleLatencyOptsOK(t testing.TB, opts LatencyOpts) {
	_, err := NewRegistry().NewLatencies(opts)
	assert.NoError(t, err, "Expected success from NewLatencies.")
	assert.NotPanics(t, func() { NewRegistry().MustLatencies(opts) }, "Expected MustLatencies to succeed.")
}

func assertSimpleLatencyOptsFail(t testing.TB, opts LatencyOpts) {
	_, err := NewRegistry().NewLatencies(opts)
	assert.Error(t, err, "Expected an error from NewLatencies.")
	assert.Panics(t, func() { NewRegistry().MustLatencies(opts) }, "Expected a panic from MustLatencies.")
}

func assertVectorLatencyOptsOK(t testing.TB, opts LatencyOpts) {
	_, err := NewRegistry().NewLatenciesVector(opts)
	assert.NoError(t, err, "Expected success from NewLatenciesVector.")
	assert.NotPanics(t, func() { NewRegistry().MustLatenciesVector(opts) }, "Expected MustLatenciesVector to succeed.")
}

func assertVectorLatencyOptsFail(t testing.TB, opts LatencyOpts) {
	_, err := NewRegistry().NewLatenciesVector(opts)
	assert.Error(t, err, "Expected an error from NewLatenciesVector.")
	assert.Panics(t, func() { NewRegistry().MustLatenciesVector(opts) }, "Expected a panic from MustLatenciesVector.")
}
