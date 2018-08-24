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

package yarpchttp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/v2/yarpctransport"
)

func TestParseTTL(t *testing.T) {
	req := &yarpctransport.Request{
		Caller:    "caller",
		Service:   "service",
		Procedure: "hello",
		Encoding:  "raw",
	}

	tests := []struct {
		ttlString string
		wantErr   error
	}{
		{ttlString: "1"},
		{
			ttlString: "-1000",
			wantErr: newInvalidTTLError(
				"service",
				"hello",
				"-1000",
			),
		},
		{
			ttlString: "not an integer",
			wantErr: newInvalidTTLError(
				"service",
				"hello",
				"not an integer",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.ttlString, func(t *testing.T) {
			ctx, cancel, err := parseTTL(context.Background(), req, tt.ttlString)
			defer cancel()

			if tt.wantErr != nil && assert.Error(t, err) {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
				_, ok := ctx.Deadline()
				assert.True(t, ok)
			}
		})
	}
}
