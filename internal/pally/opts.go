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
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

// Opts configure an individual metric or vector.
type Opts struct {
	Name           string
	Help           string
	ConstLabels    Labels
	VariableLabels []string // only meaningful for vectors
	DisableTally   bool
}

func (o Opts) describe() *prometheus.Desc {
	return prometheus.NewDesc(
		o.Name,
		o.Help,
		o.VariableLabels,
		prometheus.Labels(o.ConstLabels),
	)
}

func (o Opts) validate() error {
	if o.Name == "" {
		return errors.New("metric name must not be empty")
	}
	if !isValidTallyString(o.Name) {
		// Prometheus handles its own name validation, so we only need to check
		// Tally.
		return errors.New("names must also be Tally-compatible")
	}
	if o.Help == "" {
		return errors.New("metric help must not be empty")
	}
	for k, v := range o.ConstLabels {
		if !isValidTallyString(k) || !isValidTallyString(v) {
			return errors.New("tag names and values must also be Tally-compatible")
		}
	}
	return nil
}

func (o Opts) validateVector() error {
	if err := o.validate(); err != nil {
		return err
	}
	if len(o.VariableLabels) == 0 {
		return errors.New("vectors must have variable labels")
	}
	for _, l := range o.VariableLabels {
		if !isValidTallyString(l) {
			return errors.New("variable tag names must be Tally-compatible")
		}
	}
	return nil
}

func (o Opts) copyLabels() map[string]string {
	l := make(map[string]string, len(o.ConstLabels)+len(o.VariableLabels))
	for k, v := range o.ConstLabels {
		l[k] = v
	}
	return l
}

func isValidTallyString(s string) bool {
	// Tally allows only a subset of the characters that Prometheus does.
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 48 <= c && c <= 57: // 0-9
			continue
		case 65 <= c && c <= 90: // A-Z
			continue
		case 97 <= c && c <= 122: // a-z
			continue
		case c == '_' || c == '-':
			continue
		default:
			return false
		}
	}
	return true
}
