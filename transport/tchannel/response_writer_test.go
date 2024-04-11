// Copyright (c) 2024 Uber Technologies, Inc.
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

package tchannel

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
)

func TestResponseWriterAddHeaders(t *testing.T) {
	tests := map[string]struct {
		h                  transport.Headers
		enforceHeaderRules bool
		expErr             error
		expReservedHeader  bool
		expHeaders         transport.Headers
	}{
		"success": {
			h:          transport.NewHeaders().With("foo", "bar"),
			expHeaders: transport.NewHeaders().With("foo", "bar"),
		},
		"known-reserved-header-used-which-lead-to-error": {
			h:                 transport.NewHeaders().With(ServiceHeaderKey, "any-value"),
			expErr:            fmt.Errorf("cannot use reserved header key: %s", ServiceHeaderKey),
			expReservedHeader: true,
		},
		"unknown-reserved-header-used-which-lead-reporting-metric": {
			h:                 transport.NewHeaders().With("rpc-any", "any-value"),
			expHeaders:        transport.NewHeaders().With("rpc-any", "any-value"),
			expReservedHeader: true,
		},
		"enforce-header-rules": {
			h:                  transport.NewHeaders().With("rpc-any", "any-value"),
			enforceHeaderRules: true,
			expErr:             fmt.Errorf("header with rpc prefix is not allowed in response application headers (rpc-any was passed)"),
			expReservedHeader:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			switchEnforceHeaderRules(t, tt.enforceHeaderRules)
			rw := responseWriterImpl{}

			rw.AddHeaders(tt.h)
			if tt.expErr != nil {
				errs := multierr.Errors(rw.failedWith)
				require.Len(t, errs, 1)
				assert.Equal(t, tt.expErr, errs[0])
			} else {
				assert.NoError(t, rw.failedWith)
			}
			assert.Equal(t, tt.expReservedHeader, rw.reservedHeader)
			assert.Equal(t, tt.expHeaders, rw.headers)
		})
	}
}
