// Copyright (c) 2021 Uber Technologies, Inc.
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

package yarpcerrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestAnnotateWithError(t *testing.T) {
	tests := []struct {
		name       string
		giveErr    error
		giveFormat string
		giveArgs   []interface{}
		wantErr    error
	}{
		{
			name:       "basic",
			giveErr:    yarpcerrors.FailedPreconditionErrorf("test"),
			giveFormat: "mytest",
			wantErr:    yarpcerrors.FailedPreconditionErrorf("mytest: test"),
		},
		{
			name:       "basic with args",
			giveErr:    yarpcerrors.FailedPreconditionErrorf("test"),
			giveFormat: "mytest %s",
			giveArgs: []interface{}{
				"arg1",
			},
			wantErr: yarpcerrors.FailedPreconditionErrorf("mytest arg1: test"),
		},
		{
			name:       "unannotated",
			giveErr:    errors.New("test"),
			giveFormat: "mytest",
			wantErr:    yarpcerrors.UnknownErrorf("mytest: test"),
		},
	}
	for _, n := range tests {
		t.Run(n.name, func(t *testing.T) {
			gotErr := AnnotateWithInfo(yarpcerrors.FromError(n.giveErr), n.giveFormat, n.giveArgs...)
			assert.Equal(t, n.wantErr, gotErr)
		})
	}
}
