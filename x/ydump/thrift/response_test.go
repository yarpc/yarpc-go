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
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
)

func encodeWire(w wire.Value) []byte {
	buf := &bytes.Buffer{}
	if err := protocol.Binary.Encode(w, buf); err != nil {
		panic(fmt.Errorf("Binary.Encode(%v) failed: %v", w, err))
	}
	return buf.Bytes()
}

func encodeEnveloped(e wire.Envelope) []byte {
	buf := &bytes.Buffer{}
	if err := protocol.Binary.EncodeEnveloped(e, buf); err != nil {
		panic(fmt.Errorf("Binary.EncodeEnveloped(%v) failed: %v", e, err))
	}
	return buf.Bytes()
}

func TestResponseBytesToMap(t *testing.T) {
	funcSpecs := getFuncSpecs(t, `
    exception E {
      1: required string reason
    }
    service Test {
      void fVoid()
      string fStr()
      void fEx() throws (1: E e)
    }
  `)

	s := wire.Struct{
		Fields: []wire.Field{
			{ID: 0, Value: wire.NewValueString("foo")},
		},
	}

	tests := []struct {
		msg    string
		spec   *compile.FunctionSpec
		bs     []byte
		opts   Options
		want   map[string]interface{}
		errMsg string
	}{
		{
			msg:  "fVoid with empty result struct",
			spec: funcSpecs["fVoid"],
			bs:   encodeWire(wire.NewValueStruct(wire.Struct{})),
			want: map[string]interface{}{},
		},
		{
			msg:  "fVoid with envelope",
			spec: funcSpecs["fVoid"],
			bs: encodeEnveloped(wire.Envelope{
				Name:  "method",
				Type:  wire.Reply,
				Value: wire.NewValueStruct(wire.Struct{}),
			}),
			opts: Options{UseEnvelopes: true},
			want: map[string]interface{}{},
		},
		{
			msg:  "fStr with str result",
			spec: funcSpecs["fStr"],
			bs:   encodeWire(wire.NewValueStruct(s)),
			want: map[string]interface{}{
				"result": "foo",
			},
		},
		{
			msg:  "fEx with exception",
			spec: funcSpecs["fEx"],
			bs: encodeWire(wire.NewValueStruct(wire.Struct{
				Fields: []wire.Field{
					{ID: 1, Value: wire.NewValueStruct(wire.Struct{
						Fields: []wire.Field{{ID: 1, Value: wire.NewValueString("bar")}},
					})},
				},
			})),
			want: map[string]interface{}{
				"e": map[string]interface{}{
					"reason": "bar",
				},
			},
		},
		{
			msg:    "fVoid with str result",
			spec:   funcSpecs["fVoid"],
			bs:     encodeWire(wire.NewValueStruct(s)),
			errMsg: "got unexpected result for void method",
		},
		{
			msg:  "fStr with invalid result type",
			spec: funcSpecs["fStr"],
			bs: encodeWire(wire.NewValueStruct(wire.Struct{
				Fields: []wire.Field{
					{ID: 0, Value: wire.NewValueBool(true)},
				},
			})),
			errMsg: "failed to parse result field 0",
		},
		{
			msg:  "fEx with unknown exception",
			spec: funcSpecs["fStr"],
			bs: encodeWire(wire.NewValueStruct(wire.Struct{
				Fields: []wire.Field{
					{ID: 2, Value: wire.NewValueStruct(wire.Struct{Fields: nil})},
				},
			})),
			errMsg: "got unknown exception with ID 2",
		},
		{
			msg:    "invalid bytes",
			bs:     []byte{1, 3, 3, 7},
			errMsg: "cannot parse Thrift struct",
		},
	}

	for _, tt := range tests {
		got, err := ResponseBytesToMap(tt.spec, tt.bs, tt.opts)
		if tt.errMsg != "" {
			if assert.Error(t, err, "Expected error for %v", tt.msg) {
				assert.Contains(t, err.Error(), tt.errMsg, "Error mismatch for %v", tt.msg)
			}
			continue
		}

		if assert.NoError(t, err, "No error expected for %v", tt.msg) {
			assert.Equal(t, tt.want, got, "Unexpected result for %v", tt.msg)
		}
	}
}

func TestResponseBytesToWire(t *testing.T) {
	s := wire.Struct{
		Fields: []wire.Field{
			{ID: 0, Value: wire.NewValueString("foo")},
		},
	}

	tests := []struct {
		msg    string
		bs     []byte
		opts   Options
		want   wire.Struct
		errMsg string
	}{
		{
			msg:  "valid struct",
			bs:   encodeWire(wire.NewValueStruct(s)),
			want: s,
		},
		{
			msg: "valid struct",
			bs: encodeEnveloped(wire.Envelope{
				Name:  "method",
				Type:  wire.Reply,
				Value: wire.NewValueStruct(s),
			}),
			opts: Options{UseEnvelopes: true},
			want: s,
		},
		{
			msg:    "bool instead of struct",
			bs:     encodeWire(wire.NewValueBool(true)),
			errMsg: "cannot parse Thrift struct from response",
		},
		{
			msg:    "invalid bytes",
			bs:     []byte{1, 3, 3, 7},
			errMsg: "cannot parse Thrift struct",
		},
	}

	for _, tt := range tests {
		got, err := responseBytesToWire(tt.bs, tt.opts)
		if tt.errMsg != "" {
			if assert.Error(t, err, "Expected to fail: %s", tt.msg) {
				assert.Contains(t, err.Error(), tt.errMsg, "Error message mismatch: %s", tt.msg)
			}
			continue
		}

		if assert.NoError(t, err, "Expected not to fail: %s", tt.msg) {
			assert.Equal(t, tt.want, got, "Result mismatch: %s", tt.msg)
		}
	}
}

func TestCheckSuccess(t *testing.T) {
	funcSpecs := getFuncSpecs(t, `
    exception E {
      1: required string reason
    }
    service Test {
			void m1()
			i32 m2()
      void m1Ex() throws (1: E e)
			i32 m2Ex() throws (1: E e)
    }
  `)

	emptyResult := wire.NewValueStruct(wire.Struct{})
	onlyResult := wire.NewValueStruct(wire.Struct{Fields: []wire.Field{
		{ID: 0, Value: wire.NewValueI32(0)},
	}})
	onlyEx := wire.NewValueStruct(wire.Struct{Fields: []wire.Field{
		{ID: 1, Value: wire.NewValueStruct(wire.Struct{})},
	}})
	ex2 := wire.NewValueStruct(wire.Struct{Fields: []wire.Field{
		{ID: 2, Value: wire.NewValueStruct(wire.Struct{})},
	}})
	resultAndEx := wire.NewValueStruct(wire.Struct{Fields: []wire.Field{
		{ID: 0, Value: wire.NewValueI32(0)},
		{ID: 1, Value: wire.NewValueStruct(wire.Struct{})},
	}})

	tests := []struct {
		msg    string
		method string
		bs     []byte
		opts   Options
		errMsg string
	}{
		{
			msg:    "deserialize failure",
			method: "m1",
			bs:     []byte{1, 1},
			errMsg: "could not deserialize",
		},
		{
			msg:    "void success",
			method: "m1",
			bs:     encodeWire(emptyResult),
		},
		{
			msg:    "void success with envelope",
			method: "m1",
			bs: encodeEnveloped(wire.Envelope{
				Name:  "method",
				Type:  wire.Reply,
				Value: emptyResult,
			}),
			opts: Options{UseEnvelopes: true},
		},
		{
			msg:    "exception in envelope",
			method: "m1",
			bs: encodeEnveloped(wire.Envelope{
				Name:  "method",
				Type:  wire.Exception,
				Value: onlyEx,
			}),
			opts:   Options{UseEnvelopes: true},
			errMsg: "TApplicationException{}",
		},
		{
			msg:    "unexpected result for void method",
			method: "m1",
			bs:     encodeWire(onlyResult),
			errMsg: "void method got unexpected result",
		},
		{
			msg:    "unexpected exception for void method",
			method: "m1",
			bs:     encodeWire(onlyEx),
			errMsg: "void method got exception: unknown, method has no exceptions",
		},
		{
			msg:    "unexpected result and exception for void method",
			method: "m1",
			bs:     encodeWire(resultAndEx),
			errMsg: "void method got unexpected result",
		},
		{
			msg:    "i32 return got no result",
			method: "m2",
			bs:     encodeWire(emptyResult),
			errMsg: "method with return did not get 1 field",
		},
		{
			msg:    "i32 return got two result fields",
			method: "m2",
			bs:     encodeWire(resultAndEx),
			errMsg: "method with return did not get 1 field",
		},
		{
			msg:    "i32 return success",
			method: "m2",
			bs:     encodeWire(onlyResult),
		},
		{
			msg:    "i32 return unexpected exception",
			method: "m2",
			bs:     encodeWire(onlyEx),
			errMsg: "method with return got exception: unknown, method has no exceptions",
		},
		{
			msg:    "void with exception got empty result - success",
			method: "m1Ex",
			bs:     encodeWire(emptyResult),
		},
		{
			msg:    "void with exception got exception",
			method: "m1Ex",
			bs:     encodeWire(onlyEx),
			errMsg: "void method got exception: e E",
		},
		{
			msg:    "void with exception got result",
			method: "m1Ex",
			bs:     encodeWire(onlyResult),
			errMsg: "void method got unexpected result",
		},
		{
			msg:    "void with exception got result and exception",
			method: "m1Ex",
			bs:     encodeWire(resultAndEx),
			errMsg: "void method got unexpected result",
		},
		{
			msg:    "i32 return with exception got nothing",
			method: "m2Ex",
			bs:     encodeWire(emptyResult),
			errMsg: "method with return did not get 1 field",
		},
		{
			msg:    "i32 return with exception got result",
			method: "m2Ex",
			bs:     encodeWire(onlyResult),
		},
		{
			msg:    "i32 return with exception got unknown exception",
			method: "m2Ex",
			bs:     encodeWire(ex2),
			errMsg: "method with return got exception: unknown",
		},
		{
			msg:    "i32 return with exception got both",
			method: "m2Ex",
			bs:     encodeWire(resultAndEx),
			errMsg: "method with return did not get 1 field",
		},
	}

	for _, tt := range tests {
		err := CheckSuccess(funcSpecs[tt.method], tt.bs, tt.opts)
		if tt.errMsg == "" {
			assert.NoError(t, err, "%v: CheckSuccess should not fail", tt.msg)
			continue
		}

		if assert.Error(t, err, "%v: CheckSuccess should fail with: %v", tt.msg, tt.errMsg) {
			assert.Contains(t, err.Error(), tt.errMsg, "%v: CheckSuccess invalid error", tt.msg)
		}
	}
}
