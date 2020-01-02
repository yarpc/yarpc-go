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

package transport

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestBadRequestError(t *testing.T) {
	err := errors.New("derp")
	err = InboundBadRequestError(err)
	assert.True(t, IsBadRequestError(err))
}

func TestIsUnexpectedError(t *testing.T) {
	assert.True(t, IsUnexpectedError(yarpcerrors.Newf(yarpcerrors.CodeInternal, "")))
}

func TestIsTimeoutError(t *testing.T) {
	assert.True(t, IsTimeoutError(yarpcerrors.Newf(yarpcerrors.CodeDeadlineExceeded, "")))
}

func TestUnrecognizedProcedureError(t *testing.T) {
	err := UnrecognizedProcedureError(&Request{Service: "curly", Procedure: "nyuck"})
	assert.True(t, IsUnrecognizedProcedureError(err))
	assert.False(t, IsUnrecognizedProcedureError(errors.New("derp")))
}
