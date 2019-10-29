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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBool(t *testing.T) {
	tests := []struct {
		value   interface{}
		want    bool
		wantErr bool
	}{
		{
			value: true,
			want:  true,
		},
		{
			value: false,
			want:  false,
		},
		{
			value: "true",
			want:  true,
		},
		{
			value: "True",
			want:  true,
		},
		{
			value: "false",
			want:  false,
		},
		{
			value: "falsE",
			want:  false,
		},
		{
			value: "f",
			want:  false,
		},
		{
			value: "t",
			want:  true,
		},
		{
			value:   "",
			wantErr: true,
		},
		{
			value: int(1),
			want:  true,
		},
		{
			value: 0,
			want:  false,
		},
		{
			value:   1.0,
			wantErr: true,
		},
		{
			value:   0.0,
			wantErr: true,
		},
		{
			value:   -1,
			wantErr: true,
		},
		{
			value:   2,
			wantErr: true,
		},
		{
			value:   map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		got, err := parseBool(tt.value)
		if tt.wantErr {
			assert.Error(t, err, "parseBool(%v) should fail", tt.value)
			continue
		}
		if assert.NoError(t, err, "parseBool(%v) should not fail", tt.value) {
			assert.Equal(t, tt.want, got, "parseBool(%v) result mismatch", tt.value)
		}
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		value   interface{}
		bits    int
		want    int64
		wantErr bool
	}{
		{
			value: 0,
			bits:  8,
			want:  0,
		},
		{
			value:   1.0,
			bits:    8,
			wantErr: true,
		},
		{
			value:   true,
			bits:    8,
			wantErr: true,
		},
		{
			value: -128,
			bits:  8,
			want:  -128,
		},
		{
			value: 127,
			bits:  8,
			want:  127,
		},
		{
			// out of range
			value:   -129,
			bits:    8,
			wantErr: true,
		},
		{
			// out of range
			value:   128,
			bits:    8,
			wantErr: true,
		},
		{
			value: 256,
			bits:  16,
			want:  256,
		},
		{
			value:   65536,
			bits:    16,
			wantErr: true,
		},
		{
			value: 65536,
			bits:  32,
			want:  65536,
		},
		{
			value:   4294967296,
			bits:    32,
			wantErr: true,
		},
		{
			value: 4294967296,
			bits:  64,
			want:  4294967296,
		},
		{
			value:   uint64(18446744073709551615),
			bits:    64,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		got, err := parseInt(tt.value, tt.bits)
		if tt.wantErr {
			assert.Error(t, err, "parseInt(%v, %v) should fail", tt.value, tt.bits)
			continue
		}
		if assert.NoError(t, err, "parseInt(%v, %v) should not fail", tt.value, tt.bits) {
			assert.Equal(t, tt.want, got, "parseInt(%v, %v) result mismatch", tt.value, tt.bits)
		}
	}
}

func TestParseDouble(t *testing.T) {
	tests := []struct {
		value   interface{}
		want    float64
		wantErr bool
	}{
		{
			value: int(1),
			want:  1,
		},
		{
			value: int64(1),
			want:  1,
		},
		{
			value: uint64(1),
			want:  1,
		},
		{
			value: 0.0,
			want:  0.0,
		},
		{
			value: 3.14159,
			want:  3.14159,
		},
		{
			value:   "ast",
			wantErr: true,
		},
		{
			value:   "0",
			wantErr: true,
		},
		{
			value:   true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		got, err := parseDouble(tt.value)
		if tt.wantErr {
			assert.Error(t, err, "parseDouble(%v) should fail", tt.value)
			continue
		}
		if assert.NoError(t, err, "parseDouble(%v) should not fail", tt.value) {
			assert.Equal(t, tt.want, got, "parseDouble(%v) result mismatch", tt.value)
		}
	}
}

func TestParseBinary(t *testing.T) {
	tests := []struct {
		value  interface{}
		want   []byte
		errMsg string
	}{
		{
			value: "",
			want:  []byte(""),
		},
		{
			value: "asd",
			want:  []byte("asd"),
		},
		{
			value: []interface{}{"a", "s", "d"},
			want:  []byte("asd"),
		},
		{
			value: []interface{}{65, 66},
			want:  []byte("AB"),
		},
		{
			value: []interface{}{104, "ello", "", " ", "world"},
			want:  []byte("hello world"),
		},
		{
			value: map[interface{}]interface{}{"base64": "YWI="},
			want:  []byte("ab"),
		},
		{
			value: map[interface{}]interface{}{"base64": "YWI"},
			want:  []byte("ab"),
		},
		{
			value: map[interface{}]interface{}{"file": "./testdata/valid.json"},
			want:  []byte(`{"k1": "v1", "k2": 5}` + "\n"),
		},
		{
			value:  []interface{}{256},
			errMsg: "failed to parse list of bytes",
		},
		{
			value:  []interface{}{1.5},
			errMsg: "can only parse list of bytes",
		},
		{
			value:  map[interface{}]interface{}{"base64": true},
			errMsg: "base64 must be specified as string",
		},
		{
			value:  map[interface{}]interface{}{"base64": "a_b"},
			errMsg: "illegal base64 data",
		},
		{
			value:  map[interface{}]interface{}{"unsupported": "ab"},
			errMsg: errBinaryObjectOptions.Error(),
		},
		{
			value:  map[interface{}]interface{}{"file": true},
			errMsg: "file requires filename as string",
		},
		{
			value:  map[interface{}]interface{}{"file": "not-found.json"},
			errMsg: "no such file or directory",
		},
		{
			value: 3.14159,
			want:  []byte("3.14159"),
		},
		{
			value: true,
			want:  []byte("true"),
		},
		{
			value: 1,
			want:  []byte("1"),
		},
		{
			value:  []bool{true},
			errMsg: "cannot parse binary/string",
		},
		{
			value: []interface{}{0, 0, 0},
			want:  []byte{0, 0, 0},
		},
	}

	for _, tt := range tests {
		got, err := parseBinary(tt.value)
		if tt.errMsg != "" {
			if assert.Error(t, err, "parseBinary(%v) should fail", tt.value) {
				assert.Contains(t, err.Error(), tt.errMsg, "parseBinary(%v) unexpected error", tt.value)
			}
			continue
		}
		if assert.NoError(t, err, "parseBinary(%v) should not fail", tt.value) {
			assert.Equal(t, tt.want, got, "parseBinary(%v) result mismatch", tt.value)
		}
	}
}
