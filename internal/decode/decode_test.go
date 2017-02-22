package decode

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type stringSet map[string]struct{}

func (ss *stringSet) Decode(dec Into) error {
	var items []string
	if err := dec(&items); err != nil {
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
		SomeValue        float64 `config:"some_value"`
		Timeout          time.Duration
		PtrToTimeout     *time.Duration

		StringSet           stringSet
		PtrToStringSet      *stringSet
		PtrToPtrToStringSet **stringSet

		AlwaysFails *sadDecoder
	}

	tests := []struct {
		desc string
		give interface{}

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
			desc: "string to *string",
			give: map[interface{}]interface{}{
				"ptrToString": someString,
			},
			want: someStruct{PtrToString: ptrToString},
		},
		{
			desc: "int to string",
			give: map[string]string{"int": "42"},
			wantErrors: []string{
				"'Int' expected type 'int', got unconvertible type 'string'",
			},
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
			desc: "config tag",
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
				"error decoding 'AlwaysFails': could not decode decode.sadDecoder from struct {}: great sadness",
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
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			var dest someStruct
			err := Decode(&dest, tt.give)

			if len(tt.wantErrors) == 0 {
				assert.NoError(t, err, "expected success")
				assert.Equal(t, tt.want, dest, "result mismatch")
				return
			}

			assert.Error(t, err, "expected error")
			for _, msg := range tt.wantErrors {
				assert.Contains(t, err.Error(), msg)
			}
		})
	}
}
