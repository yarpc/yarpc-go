// Copyright (c) 2019 Uber Technologies, Inc.
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

package yarpcconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/config"
	"go.uber.org/yarpc/internal/whitespace"
	"gopkg.in/yaml.v2"
)

func TestBackoffConfig(t *testing.T) {
	type testCase struct {
		name string
		give string
		env  map[string]string
		want Backoff
		err  bool
	}

	tests := []testCase{
		{
			name: "empty",
		},
		{
			name: "specified",
			give: `
				exponential:
					first: 1s
					max: 2s
			`,
			want: Backoff{
				Exponential: ExponentialBackoff{
					First: time.Second,
					Max:   2 * time.Second,
				},
			},
		},
		{
			name: "bogus",
			give: `
				whatevenis: true
			`,
			err: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := whitespace.Expand(tt.give)
			var data map[string]interface{}
			require.NoError(t, yaml.Unmarshal([]byte(text), &data))

			var cfg Backoff
			err := config.DecodeInto(&cfg, data, config.InterpolateWith(mapVariableResolver(tt.env)))

			if err == nil {
				_, err = cfg.Strategy()
			}

			if tt.err {
				require.NotNil(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, cfg, tt.want)
			}
		})
	}
}
