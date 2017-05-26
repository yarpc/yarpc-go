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

package rereader

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ReReaderAction defines actions that can be applied to a ReReader.
type ReReaderAction interface {
	// Apply runs a function on the ReReader
	Apply(*testing.T, *ReReader)
}

// ReadAction is an action that runs a read on the ReReader.
type ReadAction struct {
	Into      []byte
	WantBytes []byte
	WantN     int
	WantError error
}

// Apply runs "Read" on the ReReader and validates the result.
func (a ReadAction) Apply(t *testing.T, rr *ReReader) {
	n, err := rr.Read(a.Into)

	assert.Equal(t, a.WantN, n)
	assert.Equal(t, a.WantError, err)
	assert.Equal(t, a.WantBytes, a.Into)
}

// ResetAction is an action that resets the ReReader.
type ResetAction struct {
	WantError error
}

// Apply runs "Reset" on the ReReader.
func (a ResetAction) Apply(t *testing.T, rr *ReReader) {
	err := rr.Reset()
	assert.Equal(t, a.WantError, err)
}

// ApplyReReaderActions runs all the ReReaderActions on the ReReader.
func ApplyReReaderActions(t *testing.T, rr *ReReader, actions []ReReaderAction) {
	for i, action := range actions {
		t.Run(fmt.Sprintf("action #%d: %T", i, action), func(t *testing.T) {
			action.Apply(t, rr)
		})
	}
}
