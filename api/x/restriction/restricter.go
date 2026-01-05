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

// Package restriction is an experimental package for preventing unwanted
// transport-encoding pairs.
//
// This package is under `x/` and subject to change. See README for details on
// 'x' packages.
package restriction

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/yarpc/api/transport"
)

// Checker is used by encoding clients, for example Protobuf and Thrift, to
// prevent unwanted transport-encoding combinations.
//
// Errors indicate whitelisted combinations.
type Checker interface {
	Check(encoding transport.Encoding, transportName string) error
}

// Tuple defines a combination to whitelist.
type Tuple struct {
	Transport string
	Encoding  transport.Encoding
}

// Validate verifes that a tuple has all fields set.
func (t Tuple) Validate() error {
	if t.Transport == "" || t.Encoding == "" {
		return errors.New("tuple missing must have all fields set")
	}
	return nil
}

// String implements fmt.Stringer.
func (t Tuple) String() string {
	return fmt.Sprintf("%s/%s", t.Transport, t.Encoding)
}

type checker struct {
	availableMsg string
	tuples       map[Tuple]struct{}
}

// NewChecker creates a Checker with a whitelist tuple combinations.
func NewChecker(tuples ...Tuple) (Checker, error) {
	if len(tuples) == 0 {
		return nil, errors.New("NewChecker requires at least one whitelisted tuple")
	}

	m := make(map[Tuple]struct{}, len(tuples))
	for _, t := range tuples {
		if err := t.Validate(); err != nil {
			return nil, err
		}
		m[t] = struct{}{}
	}

	elements := make([]string, 0, len(tuples))
	for _, t := range tuples {
		elements = append(elements, t.String())
	}

	return &checker{
		tuples:       m,
		availableMsg: strings.Join(elements, ","),
	}, nil
}

// Check returns nil for supported transport/encoding combinations and errors
// for unsupported combinations. Errors indicate whitelisted combinations.
//
// Nil Checker will alwas return nil.
func (r *checker) Check(encoding transport.Encoding, transportName string) error {
	t := Tuple{
		Transport: transportName,
		Encoding:  encoding,
	}

	if _, ok := r.tuples[t]; ok {
		return nil
	}

	return fmt.Errorf("%q is not a whitelisted combination, available: %q", t.String(), r.availableMsg)
}
