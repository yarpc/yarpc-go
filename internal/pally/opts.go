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
	"fmt"

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
	if !IsValidName(o.Name) {
		return fmt.Errorf("metric name %q is not Pally-compatible", o.Name)
	}
	if o.Help == "" {
		return errors.New("metric help must not be empty")
	}
	for k, v := range o.ConstLabels {
		if !IsValidName(k) || !IsValidLabelValue(v) {
			return fmt.Errorf("label %q=%q contains Pally-incompatible characters", k, v)
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
		if !IsValidName(l) {
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
