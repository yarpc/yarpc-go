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

package thrift

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/wire"
)

// DummyFS is an in-memory implementation of the Filesystem interface.
type DummyFS map[string][]byte

// Read returns the contents for the specified file.
func (fs DummyFS) Read(filename string) ([]byte, error) {
	if contents, ok := fs[filename]; ok {
		return contents, nil
	}
	return nil, os.ErrNotExist
}

// Abs returns the absolute path for the specified file.
// The dummy implementation always returns the original path.
func (DummyFS) Abs(filename string) (string, error) {
	return filename, nil
}

// thriftTestParse parses the given file contents as a Thrift file with no dependencies.
func thriftTestParse(t *testing.T, contents string) *compile.Module {
	fs := DummyFS{
		"file.thrift": []byte(contents),
	}
	compiled, err := compile.Compile("file.thrift", compile.Filesystem(fs))
	require.NoError(t, err, "Failed to compile thrift file:\n%s", contents)
	return compiled
}

// getFuncSpecs returns the spec for a service named "Test".
func getFuncSpecs(t *testing.T, contents string) map[string]*compile.FunctionSpec {
	module := thriftTestParse(t, contents)
	svc, err := module.LookupService("Test")
	require.NoError(t, err, "Thrift contents missing service Test")
	return svc.Functions
}

func TestParseRequest(t *testing.T) {
	funcSpec := getFuncSpecs(t, `
    struct NS {
      1: optional string f1
    }

		const S S_default = {
			"f1": "f1_val",
		}

    struct S {
      1: required string f1,
      2: optional NS ns,
      3: optional NS ns_,
    }

		union U {
			1: string s
			2: i32 i
		}

		struct SWrap {
			1: required S s = S_default
			2: optional S s2 = S_default
		}

    typedef i32 Foo

		enum Op {
			Add = 1,
			MULTIPLY,
		}

    service Test {

      void test(
        1: i8 arg1,
        2: i64 arg2,
        3: string arg3,
        4: binary arg4,
        5: S s,
        6: Foo foo,
				7: U u,
				8: SWrap sWrap,
				9: i16 arg_i16,
				10: i32 arg_i32,
				11: bool arg_bool,
				12: double arg_double,
				13: optional list<string> str_list,
				14: optional set<string> str_set,
				15: optional map<string, i32> s_i_map;
				16: optional map<i32, i32> i_i_map;
				17: optional map<bool, i32> b_i_map;
				18: optional Op op;
				19: optional list<i32> i32_list;
      )
    }

  `)["test"]
	tests := []struct {
		request map[string]interface{}
		want    []wire.Field
		errMsg  string
	}{
		{
			// No arguments is valid since arguments are not required.
			request: map[string]interface{}{},
			want:    []wire.Field{},
		},
		{
			// Unknown argument
			request: map[string]interface{}{
				"foo2": 1,
			},
			errMsg: fieldGroupError{notFound: []string{"foo2"}}.Error(),
		},
		{
			// Set numbers and bools
			request: map[string]interface{}{
				"arg1":       1,
				"arg2":       2,
				"arg_i16":    3,
				"arg_i32":    4,
				"arg_bool":   true,
				"arg_double": 5.6,
			},
			want: []wire.Field{
				{ID: 1, Value: wire.NewValueI8(1)},
				{ID: 2, Value: wire.NewValueI64(2)},
				{ID: 9, Value: wire.NewValueI16(3)},
				{ID: 10, Value: wire.NewValueI32(4)},
				{ID: 11, Value: wire.NewValueBool(true)},
				{ID: 12, Value: wire.NewValueDouble(5.6)},
			},
		},
		{
			// Set the wrong type for a number
			request: map[string]interface{}{
				"arg1": "asd",
			},
			errMsg: `field "byte" cannot parse int8 from string: asd`,
		},
		{
			// Set the typedef
			request: map[string]interface{}{
				"foo": 3,
			},
			want: []wire.Field{
				{ID: 6, Value: wire.NewValueI32(3)},
			},
		},
		{
			// Set the string and the binary field
			request: map[string]interface{}{
				"arg3": "string",
				"arg4": "binary",
			},
			want: []wire.Field{
				{ID: 3, Value: wire.NewValueString("string")},
				{ID: 4, Value: wire.NewValueBinary([]byte("binary"))},
			},
		},
		{
			// Set the struct and all nested fields
			request: map[string]interface{}{
				"s": map[string]interface{}{
					"f1": "foo",
					"ns": map[string]interface{}{
						"f1": "f1",
					},
				},
			},
			want: []wire.Field{
				{ID: 5, Value: wire.NewValueStruct(wire.Struct{
					Fields: []wire.Field{
						{ID: 1, Value: wire.NewValueString("foo")},
						{ID: 2, Value: wire.NewValueStruct(wire.Struct{
							Fields: []wire.Field{
								{ID: 1, Value: wire.NewValueString("f1")},
							},
						})}},
				})},
			},
		},
		{
			// s has a required field which is not set.
			request: map[string]interface{}{
				"s": map[string]interface{}{},
			},
			errMsg: fieldGroupError{missingRequired: []string{"f1"}}.Error(),
		},
		{
			// s has multiple errors, required field and unexpected field.
			request: map[string]interface{}{
				"s": map[string]interface{}{
					"funknown": 1,
				},
			},
			errMsg: fieldGroupError{notFound: []string{"funknown"}, missingRequired: []string{"f1"}}.Error(),
		},
		{
			// structs must be specified using map[string]interface{}
			request: map[string]interface{}{
				"s": []interface{}{"funknown"},
			},
			errMsg: errStructUseMapString.Error(),
		},
		{
			request: map[string]interface{}{
				"u": map[string]interface{}{
					"s": "foo",
				},
			},
			want: []wire.Field{
				{ID: 7, Value: wire.NewValueStruct(wire.Struct{
					Fields: []wire.Field{
						{ID: 1, Value: wire.NewValueString("foo")},
					},
				})},
			},
		},
		{
			// union cannot have 0 fields.
			request: map[string]interface{}{
				"u": map[string]interface{}{},
			},
			errMsg: errUsingSingleField.Error(),
		},
		{
			// union cannot have 2 fields.
			request: map[string]interface{}{
				"u": map[string]interface{}{
					"s": "foo",
					"i": 1,
				},
			},
			errMsg: errUsingSingleField.Error(),
		},
		{
			// sWrap has a required field with a default.
			request: map[string]interface{}{
				"sWrap": map[string]interface{}{},
			},
			want: []wire.Field{
				{ID: 8, Value: wire.NewValueStruct(wire.Struct{
					Fields: []wire.Field{
						{ID: 1, Value: wire.NewValueStruct(wire.Struct{
							Fields: []wire.Field{
								{ID: 1, Value: wire.NewValueString("f1_val")},
							},
						})},
						{ID: 2, Value: wire.NewValueStruct(wire.Struct{
							Fields: []wire.Field{
								{ID: 1, Value: wire.NewValueString("f1_val")},
							},
						})},
					},
				})},
			},
		},
		{
			request: map[string]interface{}{
				"str_list": []interface{}{"a", "b", "c"},
			},
			want: []wire.Field{{
				ID: 13,
				Value: wire.NewValueList(wire.ValueListFromSlice(
					wire.TBinary,
					[]wire.Value{
						wire.NewValueString("a"),
						wire.NewValueString("b"),
						wire.NewValueString("c"),
					},
				)),
			}},
		},
		{
			// wrong type for list
			request: map[string]interface{}{
				"str_list": "asd",
			},
			errMsg: "must be specified using list[*]",
		},
		{
			request: map[string]interface{}{
				"i32_list": []interface{}{1, "a"},
			},
			errMsg: "list item failed",
		},
		{
			request: map[string]interface{}{
				"str_set": []interface{}{"a", "b", "c"},
			},
			want: []wire.Field{{
				ID: 14,
				Value: wire.NewValueSet(wire.ValueListFromSlice(
					wire.TBinary,
					[]wire.Value{
						wire.NewValueString("a"),
						wire.NewValueString("b"),
						wire.NewValueString("c"),
					},
				)),
			}},
		},
		{
			request: map[string]interface{}{
				"s_i_map": map[string]interface{}{
					"a": 1,
					"b": 2,
					"c": 3,
				},
			},
			want: []wire.Field{{
				ID: 15,
				Value: wire.NewValueMap(wire.MapItemListFromSlice(
					wire.TBinary,
					wire.TI32,
					[]wire.MapItem{
						{wire.NewValueString("a"), wire.NewValueI32(1)},
						{wire.NewValueString("b"), wire.NewValueI32(2)},
						{wire.NewValueString("c"), wire.NewValueI32(3)},
					},
				)),
			}},
		},
		{
			request: map[string]interface{}{
				"i_i_map": map[interface{}]interface{}{
					2: 1,
					1: 2,
				},
			},
			want: []wire.Field{{
				ID: 16,
				Value: wire.NewValueMap(wire.MapItemListFromSlice(
					wire.TI32,
					wire.TI32,
					[]wire.MapItem{
						{wire.NewValueI32(2), wire.NewValueI32(1)},
						{wire.NewValueI32(1), wire.NewValueI32(2)},
					},
				)),
			}},
		},
		{
			request: map[string]interface{}{
				"i_i_map": map[interface{}]interface{}{
					"2": 1,
					"1": 2,
				},
			},
			want: []wire.Field{{
				ID: 16,
				Value: wire.NewValueMap(wire.MapItemListFromSlice(
					wire.TI32,
					wire.TI32,
					[]wire.MapItem{
						{wire.NewValueI32(2), wire.NewValueI32(1)},
						{wire.NewValueI32(1), wire.NewValueI32(2)},
					},
				)),
			}},
		},
		{
			request: map[string]interface{}{
				"op": 1,
			},
			want: []wire.Field{
				{ID: 18, Value: wire.NewValueI32(1)},
			},
		},
		{
			request: map[string]interface{}{
				"op": "Op(999)",
			},
			want: []wire.Field{
				{ID: 18, Value: wire.NewValueI32(999)},
			},
		},
		{
			request: map[string]interface{}{
				"op": "Add",
			},
			want: []wire.Field{
				{ID: 18, Value: wire.NewValueI32(1)},
			},
		},
		{
			request: map[string]interface{}{
				"op": "Multiply",
			},
			want: []wire.Field{
				{ID: 18, Value: wire.NewValueI32(2)},
			},
		},
		{
			// map is not the right type.
			request: map[string]interface{}{
				"s_i_map": "asd",
			},
			errMsg: errMapUnknownType.Error(),
		},
		{
			// map key type is wrong.
			request: map[string]interface{}{
				"b_i_map": map[string]interface{}{
					"asd": 1,
				},
			},
			errMsg: "map key (asd) failed",
		},
		{
			// map value type is wrong.
			request: map[string]interface{}{
				"b_i_map": map[string]interface{}{
					"true": "asd",
				},
			},
			errMsg: "map value (asd) for key (true) failed",
		},
		{
			// use fuzzy matching to set fields.
			request: map[string]interface{}{
				"Arg1":    1,
				"argi16_": 3,
				"ARGi32":  4,
				"ArgBool": true,
			},
			want: []wire.Field{
				{ID: 1, Value: wire.NewValueI8(1)},
				{ID: 9, Value: wire.NewValueI16(3)},
				{ID: 10, Value: wire.NewValueI32(4)},
				{ID: 11, Value: wire.NewValueBool(true)},
			},
		},
		{
			// fuzzy matching is disabled on any name clashes.
			request: map[string]interface{}{
				"s": map[string]interface{}{
					"f1": "foo",
					"NS": "asd",
				},
			},
			errMsg: fieldGroupError{notFound: []string{"NS"}}.Error(),
		},
		{
			// allow specifying fields by field ID
			request: map[string]interface{}{
				"1":  1,
				"9":  3,
				"11": true,
			},
			want: []wire.Field{
				{ID: 1, Value: wire.NewValueI8(1)},
				{ID: 9, Value: wire.NewValueI16(3)},
				{ID: 11, Value: wire.NewValueBool(true)},
			},
		},
		{
			// invalid field IDs should report an error though.
			request: map[string]interface{}{
				"999": 1,
			},
			errMsg: fieldGroupError{notFound: []string{"999"}}.Error(),
		},
		{
			// Unknown enum
			request: map[string]interface{}{
				"op": "Divide",
			},
			errMsg: `unrecognized enum "Divide"`,
		},
		{
			// Unknown enum
			request: map[string]interface{}{
				"op": "Op(NaN)",
			},
			errMsg: `unrecognized enum "Op(NaN)"`,
		},
	}

	for _, tt := range tests {
		req := map[string]interface{}(tt.request)
		got, err := structToValue(compile.FieldGroup(funcSpec.ArgsSpec), req)
		if tt.errMsg != "" {
			if assert.Error(t, err, "Expected error for %v", req) {
				assert.Contains(t, err.Error(), tt.errMsg, "Unexpected error for %v", req)
			}
			continue
		}
		if assert.NoError(t, err, "Expected no error for %v", req) {
			want := wire.NewValueStruct(wire.Struct{Fields: tt.want})
			assert.True(t, wire.ValuesAreEqual(want, wire.NewValueStruct(got)),
				"Parsed mismatch for %v:\n want %v\n  got %v\n", req, want, got)
		}
	}
}

func TestParseSuccess(t *testing.T) {
	tempFile, err := ioutil.TempFile("", "thrift")
	require.NoError(t, err, "Failed to get a temporary file")
	filePrefix := tempFile.Name() + "_1"
	fullFile := filePrefix + ".thrift"

	data := []byte(`
		struct S {
			1: string s
		}
	`)
	require.NoError(t, ioutil.WriteFile(fullFile, data, 0666),
		"Failed to write thrift file to %v", fullFile)
	defer os.Remove(fullFile)

	for _, fname := range []string{fullFile, filePrefix} {
		module, err := Parse(fname)
		if assert.NoError(t, err, "Parse(%v) failed", fname) {
			assert.NotNil(t, module, "No module was returned")
		}
	}
}

func TestSplitMethod(t *testing.T) {
	tests := []struct {
		fullMethod string
		wantSvc    string
		wantMethod string
		wantErr    bool
	}{
		{
			fullMethod: "",
			wantSvc:    "",
			wantMethod: "",
		},
		{
			fullMethod: "Svc",
			wantSvc:    "Svc",
			wantMethod: "",
		},
		{
			fullMethod: "Svc::",
			wantSvc:    "Svc",
			wantMethod: "",
		},
		{
			fullMethod: "Svc::Method",
			wantSvc:    "Svc",
			wantMethod: "Method",
		},
		{
			fullMethod: "Svc::Method::Something",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		svc, method, err := SplitMethod(tt.fullMethod)
		if tt.wantErr {
			assert.Error(t, err, "SplitMethod(%v) should fail", tt.fullMethod)
			continue
		}

		if assert.NoError(t, err, "SplitMethod(%v) failed unexpectedly", tt.fullMethod) {
			assert.Equal(t, tt.wantSvc, svc, "SplitMethod(%v) got svc mismatch", tt.fullMethod)
			assert.Equal(t, tt.wantMethod, method, "SplitMethod(%v) got method mismatch", tt.fullMethod)
		}
	}
}

func TestRequestToBytes(t *testing.T) {
	funcSpec := getFuncSpecs(t, `
		service Test {
			void test(1: string s)
		}
	`)["test"]

	tests := []struct {
		request map[string]interface{}
		opts    Options
		wantErr bool
	}{
		{
			request: map[string]interface{}{
				"s": "foo",
			},
			opts: Options{},
		},
		{
			request: map[string]interface{}{
				"s": "foo",
			},
			opts: Options{UseEnvelopes: true},
		},
		{
			request: map[string]interface{}{
				"t": "foo",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		_, err := RequestToBytes(funcSpec, tt.request, tt.opts)
		assert.Equal(t, tt.wantErr, err != nil, "wantErr %v for %v", tt.wantErr, tt.request)
	}
}
