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

package mapdecode

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stringSetYAML map[string]struct{}

func (ss *stringSetYAML) UnmarshalYAML(f func(interface{}) error) error {
	var items []string
	if err := f(&items); err != nil {
		return err
	}

	result := make(stringSetYAML)
	for _, item := range items {
		result[item] = struct{}{}
	}
	*ss = result

	return nil
}

func TestYAML(t *testing.T) {
	type value struct {
		Int       int64         `yaml:"i"`
		String    string        `yaml:"s"`
		StringSet stringSetYAML `yaml:"ss"`

		// sadDecoder implements Decode and always fails. We expect to never
		// actually call that because we're using the YAML option.
		Empty *sadDecoder `yaml:"empty" mapdecode:"notempty"`
	}

	type item struct {
		Key   string `yaml:"name"`
		Value value  `yaml:"value"`
	}

	tests := []struct {
		desc string
		give interface{}
		want item
	}{
		{
			desc: "string value",
			give: map[string]interface{}{
				"name":  "foo",
				"value": map[string]interface{}{"s": "bar"},
			},
			want: item{Key: "foo", Value: value{String: "bar"}},
		},
		{
			desc: "int value",
			give: map[string]interface{}{
				"name":  "hi",
				"value": map[string]interface{}{"i": 42},
			},
			want: item{Key: "hi", Value: value{Int: 42}},
		},
		{
			desc: "string set value",
			give: map[string]interface{}{
				"name": "zzz",
				"value": map[string]interface{}{
					"ss": []interface{}{"foo", "bar", "baz"},
				},
			},
			want: item{
				Key: "zzz",
				Value: value{
					StringSet: stringSetYAML{
						"foo": struct{}{},
						"bar": struct{}{},
						"baz": struct{}{},
					},
				},
			},
		},
		{
			desc: "sad decoder",
			give: map[string]interface{}{
				"name": "abc",
				"value": map[string]interface{}{
					"empty": map[string]interface{}{},
				},
			},
			want: item{
				Key:   "abc",
				Value: value{Empty: &sadDecoder{}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var dest item
			require.NoError(t, Decode(&dest, tt.give, YAML()))
			assert.Equal(t, tt.want, dest)
		})
	}
}
