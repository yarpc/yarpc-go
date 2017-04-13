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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stringSet map[string]struct{}

func (ss *stringSet) Decode(into Into) error {
	var items []string
	if err := into(&items); err != nil {
		return err
	}

	result := make(stringSet)
	for _, item := range items {
		result[item] = struct{}{}
	}
	*ss = result

	return nil
}

type sadDecoder struct{}

func (*sadDecoder) Decode(Into) error {
	return errors.New("great sadness")
}

func TestDecode(t *testing.T) {
	someInt := 42
	someBool := true
	ptrToInt := &someInt

	someString := "hello world"
	ptrToString := &someString

	someStringSet := stringSet{"hello": {}, "world": {}}
	ptrToStringSet := &someStringSet

	someTimeout := 4*time.Second + 2*time.Millisecond

	type someStruct struct {
		Int              int
		PtrToString      *string
		PtrToPtrToString **string
		SomeValue        float64 `mapdecode:"some_value"`
		Timeout          time.Duration
		PtrToTimeout     *time.Duration

		StringSet           stringSet
		PtrToStringSet      *stringSet
		PtrToPtrToStringSet **stringSet

		SomeBool    bool
		PtrToBool   *bool
		SomeFloat32 float32
		Unsigned    uint8

		AlwaysFails *sadDecoder
	}

	tests := []struct {
		desc string
		give interface{}
		opts []Option

		want       someStruct
		wantErrors []string
	}{
		{
			desc: "nil",
			give: nil,
		},
		{
			desc: "nil value",
			give: map[interface{}]interface{}{"int": nil},
		},
		{
			desc: "int to int",
			give: map[string]int{"int": someInt},
			want: someStruct{Int: someInt},
		},
		{
			desc: "*int to int",
			give: map[interface{}]interface{}{
				"int": ptrToInt,
			},
			want: someStruct{Int: someInt},
		},
		{
			desc: "nil *int to int",
			give: map[interface{}]interface{}{
				"int": (*int)(nil),
			},
			want: someStruct{Int: 0},
		},
		{
			desc: "string to *string",
			give: map[interface{}]interface{}{
				"ptrToString": someString,
			},
			want: someStruct{PtrToString: ptrToString},
		},
		{
			desc: "int to string",
			give: map[string]string{"int": "42"},
			want: someStruct{Int: 42},
		},
		{
			desc: "**int to int",
			give: map[interface{}]interface{}{"int": &ptrToInt},
			want: someStruct{Int: someInt},
		},
		{
			desc: "string to **string",
			give: map[interface{}]interface{}{"ptrToPtrToString": someString},
			want: someStruct{PtrToPtrToString: &ptrToString},
		},
		{
			desc: "tag",
			give: map[string]interface{}{"some_value": 42.0},
			want: someStruct{SomeValue: 42.0},
		},
		{
			desc: "stringSet",
			give: map[string]interface{}{"stringSet": []string{"hello", "world"}},
			want: someStruct{StringSet: someStringSet},
		},
		{
			desc: "*stringSet",
			give: map[string]interface{}{"ptrToStringSet": []string{"hello", "world"}},
			want: someStruct{PtrToStringSet: ptrToStringSet},
		},
		{
			desc: "**stringSet",
			give: map[interface{}]interface{}{"ptrToPtrToStringSet": []string{"hello", "world"}},
			want: someStruct{PtrToPtrToStringSet: &ptrToStringSet},
		},
		{
			desc: "decode failure",
			give: map[interface{}]interface{}{"alwaysFails": struct{}{}},
			wantErrors: []string{
				"error decoding 'AlwaysFails': could not decode mapdecode.sadDecoder from struct {}: great sadness",
			},
		},
		{
			desc: "time.Duration",
			give: map[interface{}]interface{}{"timeout": "4s2ms"},
			want: someStruct{Timeout: someTimeout},
		},
		{
			desc: "*time.Duration",
			give: map[interface{}]interface{}{"ptrToTimeout": "4s2ms"},
			want: someStruct{PtrToTimeout: &someTimeout},
		},
		{
			desc:       "unused",
			give:       map[string]interface{}{"foo": "bar"},
			wantErrors: []string{"invalid keys: foo"},
		},
		{
			desc: "ignore unused",
			give: map[string]interface{}{"foo": "bar"},
			opts: []Option{IgnoreUnused(true)},
		},
		{
			desc: "bool",
			give: map[string]interface{}{"someBool": "true"},
			want: someStruct{SomeBool: someBool},
		},
		{
			desc: "*bool",
			give: map[string]interface{}{"ptrToBool": "true"},
			want: someStruct{PtrToBool: &someBool},
		},
		{
			desc: "float32",
			give: map[string]interface{}{"someFloat32": "42.123"},
			want: someStruct{SomeFloat32: 42.123},
		},
		{
			desc: "uint8",
			give: map[string]interface{}{"unsigned": "42"},
			want: someStruct{Unsigned: 42},
		},
		{
			desc:       "uint8 overflow",
			give:       map[string]interface{}{"unsigned": "12893721983721987321"},
			wantErrors: []string{"error decoding 'Unsigned':", "value out of range"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var dest someStruct
			err := Decode(&dest, tt.give, tt.opts...)

			if len(tt.wantErrors) == 0 {
				require.NoError(t, err, "expected success")
				assert.Equal(t, tt.want, dest, "result mismatch")
				return
			}

			require.Error(t, err, "expected error")
			for _, msg := range tt.wantErrors {
				assert.Contains(t, err.Error(), msg)
			}
		})
	}
}
