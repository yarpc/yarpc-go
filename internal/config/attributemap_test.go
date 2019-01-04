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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttributeMapPopBool(t *testing.T) {
	tests := []struct {
		desc     string
		give     AttributeMap
		giveName string

		want       bool
		wantErrors []string
	}{
		{
			desc: "true decode",
			give: AttributeMap{
				"true": true,
			},
			giveName: "true",
			want:     true,
		},
		{
			desc: "false decode",
			give: AttributeMap{
				"false": false,
			},
			giveName: "false",
			want:     false,
		},
		{
			desc: "invalid decode",
			give: AttributeMap{
				"test": "teeeeeest",
			},
			giveName:   "test",
			wantErrors: []string{"failed to read attribute", `"test"`, `teeeeeest`},
		},
		{
			desc: "named string decode",
			give: AttributeMap{
				"test": "true",
			},
			giveName: "test",
			want:     true,
		},
		{
			desc: "named string decode false",
			give: AttributeMap{
				"test": "false",
			},
			giveName: "test",
			want:     false,
		},
		{
			desc:     "missing decode",
			give:     AttributeMap{},
			giveName: "test",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			res, err := tt.give.PopBool(tt.giveName)

			if len(tt.wantErrors) > 0 {
				require.Error(t, err)
				for _, msg := range tt.wantErrors {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.want, res)
		})
	}
}

func TestAttributeMapPopString(t *testing.T) {
	tests := []struct {
		desc     string
		give     AttributeMap
		giveName string

		want       string
		wantErrors []string
	}{
		{
			desc: "value decode",
			give: AttributeMap{
				"test": "value",
			},
			giveName: "test",
			want:     "value",
		},
		{
			desc: "invalid decode",
			give: AttributeMap{
				"test": []byte("value"),
			},
			giveName:   "test",
			wantErrors: []string{`failed to read attribute "test"`},
		},
		{
			desc:     "invalid name",
			give:     AttributeMap{},
			giveName: "test",
			want:     "", // we default to the Nil value
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			res, err := tt.give.PopString(tt.giveName)

			if len(tt.wantErrors) > 0 {
				require.Error(t, err)
				for _, msg := range tt.wantErrors {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.want, res)
		})
	}
}

func TestAttributeMapPop(t *testing.T) {
	type someStruct struct {
		SomeString    string `config:",interpolate"`
		AnotherString string
	}

	tests := []struct {
		desc     string
		give     AttributeMap
		giveName string
		giveInto interface{}

		env map[string]string

		want        interface{}
		wantMissing bool
		wantErrors  []string
	}{
		{
			desc: "bad string",
			give: AttributeMap{
				"test": map[string]interface{}{
					"someString": "hello ${NAME:world",
				},
			},
			giveInto: &someStruct{},
			giveName: "test",
			wantErrors: []string{
				`failed to read attribute "test"`,
				`failed to parse "hello ${NAME:world" for interpolation`,
			},
		},
		{
			desc: "bad string uninterpolated field",
			give: AttributeMap{
				"test": map[string]interface{}{
					"anotherString": "hello ${NAME:world",
				},
			},
			giveInto: &someStruct{},
			giveName: "test",
			want:     &someStruct{AnotherString: "hello ${NAME:world"},
		},
		{
			desc: "string interpolation",
			give: AttributeMap{
				"test": map[string]interface{}{
					"someString": "hello ${NAME:world}",
				},
			},
			giveInto: &someStruct{},
			giveName: "test",
			env:      map[string]string{"NAME": "foo"},
			want:     &someStruct{SomeString: "hello foo"},
		},
		{
			desc:        "missing field",
			give:        AttributeMap{},
			giveInto:    &someStruct{},
			giveName:    "test",
			want:        &someStruct{},
			wantMissing: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ok, err := tt.give.Pop(tt.giveName, tt.giveInto, InterpolateWith(mapVariableResolver(tt.env)))

			if len(tt.wantErrors) > 0 {
				require.Error(t, err)
				for _, msg := range tt.wantErrors {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantMissing, !ok)
			assert.Equal(t, tt.want, tt.giveInto)
		})
	}
}

func TestAttributeMapKeys(t *testing.T) {
	tests := []struct {
		desc string
		give AttributeMap

		wantKeys []string
	}{
		{
			desc:     "no keys",
			give:     AttributeMap{},
			wantKeys: []string{},
		},
		{
			desc: "one key",
			give: AttributeMap{
				"test": map[string]interface{}{
					"anotherString": "hello ${NAME:world",
				},
			},
			wantKeys: []string{"test"},
		},
		{
			desc: "multi keys",
			give: AttributeMap{
				"test": map[string]interface{}{
					"anotherString": "hello ${NAME:world",
				},
				"test1": true,
				"test2": "teeeeest",
				"test3": 1234,
			},
			wantKeys: []string{"test", "test1", "test2", "test3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			keys := tt.give.Keys()

			require.Equal(t, len(tt.wantKeys), len(keys))

			keyMap := make(map[string]struct{}, len(tt.wantKeys))
			for _, key := range keys {
				keyMap[key] = struct{}{}
			}

			for _, wantKey := range tt.wantKeys {
				_, ok := keyMap[wantKey]
				assert.True(t, ok, "key %q was not in map", wantKey)
			}
		})
	}
}

func TestAttributeMapDecode(t *testing.T) {
	type someStruct struct {
		SomeString    string `config:",interpolate"`
		AnotherString string
	}

	tests := []struct {
		desc     string
		give     AttributeMap
		giveInto interface{}

		env map[string]string

		want       interface{}
		wantErrors []string
	}{
		{
			desc: "bad string",
			give: AttributeMap{
				"someString": "hello ${NAME:world",
			},
			giveInto: &someStruct{},
			wantErrors: []string{
				`failed to parse "hello ${NAME:world" for interpolation`,
			},
		},
		{
			desc: "bad string uninterpolated field",
			give: AttributeMap{
				"anotherString": "hello ${NAME:world",
			},
			giveInto: &someStruct{},
			want:     &someStruct{AnotherString: "hello ${NAME:world"},
		},
		{
			desc: "string interpolation",
			give: AttributeMap{
				"someString": "hello ${NAME:world}",
			},
			giveInto: &someStruct{},
			env:      map[string]string{"NAME": "foo"},
			want:     &someStruct{SomeString: "hello foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := tt.give.Decode(tt.giveInto, InterpolateWith(mapVariableResolver(tt.env)))

			if len(tt.wantErrors) > 0 {
				require.Error(t, err)
				for _, msg := range tt.wantErrors {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.want, tt.giveInto)
		})
	}
}
