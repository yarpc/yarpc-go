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

package ioutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// RereaderAction defines actions that can be applied to a Rereader.
type RereaderAction interface {
	// Apply runs a function on the Rereader
	Apply(*testing.T, *Rereader)
}

// ReadAction is an action that runs a read on the Rereader.
type ReadAction struct {
	Into      []byte
	WantBytes []byte
	WantN     int
	WantError error
}

// Apply runs "Read" on the Rereader and validates the result.
func (a ReadAction) Apply(t *testing.T, rr *Rereader) {
	n, err := rr.Read(a.Into)

	assert.Equal(t, a.WantN, n)
	assert.Equal(t, a.WantError, err)
	assert.Equal(t, a.WantBytes, a.Into)
}

// ResetAction is an action that resets the Rereader.
type ResetAction struct {
	WantError error
}

// Apply runs "Reset" on the Rereader.
func (a ResetAction) Apply(t *testing.T, rr *Rereader) {
	err := rr.Reset()
	assert.Equal(t, a.WantError, err)
}

// ApplyRereaderActions runs all the RereaderActions on the Rereader.
func ApplyRereaderActions(t *testing.T, rr *Rereader, actions []RereaderAction) {
	for i, action := range actions {
		t.Run(fmt.Sprintf("action #%d: %T", i, action), func(t *testing.T) {
			action.Apply(t, rr)
		})
	}
}
