// Copyright (c) 2017 Uber Technologies, Inc.
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
	"regexp"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
)

func TestValidateTallyTags(t *testing.T) {
	// Regexp is a slower, but more easily verifiable, description of the Tally
	// name specification.
	tallyRe := regexp.MustCompile(`^[0-9A-z_\-]+$`)
	assert.NoError(t, quick.CheckEqual(
		isValidTallyString,
		func(s string) bool { return tallyRe.MatchString(s) },
		nil, /* config */
	), "Tally validation doesn't match regexp implementation.")
}

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
