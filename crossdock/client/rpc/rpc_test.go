// Copyright (c) 2016 Uber Technologies, Inc.
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

package rpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yarpc/yarpc-go/crossdock-go"
)

func TestCreate(t *testing.T) {
	tests := []struct {
		params crossdock.Params
		errOut string
	}{
		{
			crossdock.Params{"server": "localhost"},
			`unknown transport ""`,
		},
		{
			crossdock.Params{"transport": "http"},
			"server is required",
		},
		{
			crossdock.Params{"server": "localhost", "transport": "foo"},
			`unknown transport "foo"`,
		},
		{
			params: crossdock.Params{
				"server":    "localhost",
				"transport": "http",
			},
		},
		{
			params: crossdock.Params{
				"server":    "localhost",
				"transport": "tchannel",
			},
		},
	}

	for _, tt := range tests {
		entries := crossdock.Run(tt.params, func(ct crossdock.T) {
			rpc := Create(ct)

			// should get here only if the request succeeded
			ch := rpc.Channel("yarpc-test")
			assert.Equal(t, "client", ch.Caller)
			assert.Equal(t, "yarpc-test", ch.Service)
		})

		if tt.errOut != "" && assert.Len(t, entries, 1) {
			e := entries[0]
			assert.Equal(t, crossdock.Failed, e.Status())
			assert.Contains(t, e.Output(), tt.errOut)
		}
	}
}
