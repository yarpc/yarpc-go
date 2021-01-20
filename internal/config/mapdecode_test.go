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

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/interpolate"
)

func TestInterpolateHook(t *testing.T) {
	type someStruct struct {
		SomeString   string        `config:",interpolate"`
		SomeInt      int           `config:",interpolate"`
		SomeFloat    float32       `config:",interpolate"`
		SomeDuration time.Duration `config:",interpolate"`

		Key string `config:"name,interpolate"`

		// We don't support interpolation for lists and maps yet.
		SomeMap  map[string]string `config:",interpolate"`
		SomeList []string          `config:",interpolate"`

		// Uninterpolated fields
		AnotherString string
	}

	tests := []struct {
		desc string
		give interface{}
		env  map[string]string

		want       someStruct
		wantErrors []string
	}{
		{
			desc: "bad string",
			give: map[string]interface{}{
				"someString": "hello ${NAME:world",
			},
			wantErrors: []string{`failed to parse "hello ${NAME:world" for interpolation`},
		},
		{
			desc: "bad string uninterpolated field",
			give: map[string]interface{}{
				"anotherString": "hello ${NAME:world",
			},
			want: someStruct{AnotherString: "hello ${NAME:world"},
		},
		{
			desc: "missing variable",
			give: map[string]interface{}{"someString": "hello ${NAME}"},
			wantErrors: []string{
				`failed to render "hello ${NAME}" with environment variables`,
			},
		},
		{
			desc: "string",
			give: map[string]interface{}{
				"someString": "hello ${NAME:world}",
			},
			env:  map[string]string{"NAME": "foo"},
			want: someStruct{SomeString: "hello foo"},
		},
		{
			desc: "string default",
			give: map[string]interface{}{
				"someString": "hello ${NAME:world}",
			},
			want: someStruct{SomeString: "hello world"},
		},
		{
			desc: "int",
			give: map[string]interface{}{"someInt": "12${FOO:}5"},
			env:  map[string]string{"FOO": "34"},
			want: someStruct{SomeInt: 12345},
		},
		{
			desc: "int default",
			give: map[string]interface{}{"someInt": "12${FOO:}5"},
			want: someStruct{SomeInt: 125},
		},
		{
			desc: "float",
			give: map[string]interface{}{"SOMEFLOAT": "12${BAR:}.${BAZ:0}"},
			env:  map[string]string{"BAR": "34", "BAZ": "56"},
			want: someStruct{SomeFloat: 1234.56},
		},
		{
			desc: "float default",
			give: map[string]interface{}{"SOMEFLOAT": "12${BAR:}.${BAZ:0}"},
			want: someStruct{SomeFloat: 12},
		},
		{
			desc: "duration",
			give: map[string]interface{}{"someDuration": "5${UNIT:s}"},
			env:  map[string]string{"UNIT": "m"},
			want: someStruct{SomeDuration: 5 * time.Minute},
		},
		{
			desc: "duration default",
			give: map[string]interface{}{"someDuration": "5${UNIT:s}"},
			want: someStruct{SomeDuration: 5 * time.Second},
		},
		{
			desc: "name override",
			give: map[string]interface{}{"name": "foo ${BAR:baz}"},
			env:  map[string]string{"BAR": "foo"},
			want: someStruct{Key: "foo foo"},
		},
		{
			desc: "name override default",
			give: map[string]interface{}{"name": "foo ${BAR:baz}"},
			want: someStruct{Key: "foo baz"},
		},
		{
			desc: "uninterpolated field",
			give: map[string]interface{}{"AnotherString": "hello ${NAME}"},
			env:  map[string]string{"NAME": "world"},
			want: someStruct{AnotherString: "hello ${NAME}"},
		},
		{
			desc: "map",
			give: map[string]interface{}{
				"someMap": map[string]interface{}{
					"foo": "${BAR}",
					"baz": "${QUX}",
				},
			},
			env: map[string]string{
				"BAR": "foo",
				"QUX": "baz",
			},
			want: someStruct{
				SomeMap: map[string]string{
					"foo": "${BAR}",
					"baz": "${QUX}",
				},
			},
		},
		{
			desc: "list",
			give: map[string]interface{}{
				"someList": []interface{}{
					"foo",
					"${BAR}",
					"baz",
					"${QUX}",
				},
			},
			env: map[string]string{
				"BAR": "foo",
				"QUX": "baz",
			},
			want: someStruct{
				SomeList: []string{
					"foo",
					"${BAR}",
					"baz",
					"${QUX}",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var dest someStruct
			err := DecodeInto(&dest, tt.give, InterpolateWith(mapVariableResolver(tt.env)))

			if len(tt.wantErrors) > 0 {
				require.Error(t, err)
				for _, msg := range tt.wantErrors {
					assert.Contains(t, err.Error(), msg)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, dest)
		})
	}
}

func mapVariableResolver(m map[string]string) interpolate.VariableResolver {
	return func(name string) (value string, ok bool) {
		value, ok = m[name]
		return
	}
}
