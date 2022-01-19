// Copyright (c) 2022 Uber Technologies, Inc.
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

package errorsync

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorWaiter(t *testing.T) {
	one := errors.New("1")
	two := errors.New("2")

	tests := []struct {
		desc string
		errs []error
		want []error
	}{
		{
			"nothing",
			nil,
			nil,
		},
		{
			"empty list",
			[]error{},
			nil,
		},
		{
			"no errors",
			[]error{nil, nil, nil},
			nil,
		},
		{
			"single error",
			[]error{nil, one, nil},
			[]error{one},
		},
		{
			"multiple errors",
			[]error{nil, one, two, nil},
			[]error{one, two},
		},
	}

	for _, tt := range tests {
		want := make(map[error]struct{})
		for _, err := range tt.want {
			want[err] = struct{}{}
		}

		var ew ErrorWaiter
		for _, err := range tt.errs {
			// Need to create a local variable to make sure that the correct
			// value is used by the closure since the value 'err' points to
			// will change between iterations.
			errLocal := err
			ew.Submit(func() error { return errLocal })
		}

		got := make(map[error]struct{})
		for _, err := range ew.Wait() {
			got[err] = struct{}{}
		}

		assert.Equal(t, want, got, tt.desc)
	}
}
