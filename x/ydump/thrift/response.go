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

	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/envelope"
	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
)

type encodedException struct {
	error
}

// ResponseBytesToMap takes the given response bytes and creates a map that
// uses field name as keys.
func ResponseBytesToMap(spec *compile.FunctionSpec, responseBytes []byte, opts Options) (map[string]interface{}, error) {
	w, err := responseBytesToWire(responseBytes, opts)
	if err != nil {
		return nil, err
	}

	var specs map[int16]*compile.FieldSpec
	if spec.ResultSpec != nil {
		specs = getFieldMap(spec.ResultSpec.Exceptions)
	}

	result := make(map[string]interface{})
	for _, f := range w.Fields {
		err = nil
		if f.ID == 0 {
			// Field ID 0 is always the result.
			if spec.ResultSpec == nil || spec.ResultSpec.ReturnType == nil {
				return nil, fmt.Errorf("got unexpected result for void method: %v", f.Value)
			}
			result["result"], err = FromWireValue(spec.ResultSpec.ReturnType, f.Value)
		} else {
			exSpec, ok := specs[f.ID]
			if !ok {
				return nil, fmt.Errorf("got unknown exception with ID %v: %v", f.ID, f.Value)
			}

			result[exSpec.Name], err = FromWireValue(exSpec.Type, f.Value)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse result field %v: %v", f.ID, err)
		}
	}

	return result, nil
}

func checkException(spec *compile.FunctionSpec, fieldID int16) string {
	if spec.ResultSpec == nil || len(spec.ResultSpec.Exceptions) == 0 {
		return "unknown, method has no exceptions"
	}
	for _, ex := range spec.ResultSpec.Exceptions {
		if ex.ID == fieldID {
			return ex.ThriftName() + " " + ex.Type.ThriftName()
		}
	}

	return "unknown"
}

// CheckSuccess returns an error if the result is not successful.
// A response is successful if:
// - Thrift deserialization is successful (lazy fields are not evaluated)
// - Only Field ID 0 (if the method has a return type) or no fields are set.
func CheckSuccess(spec *compile.FunctionSpec, responseBytes []byte, opts Options) error {
	w, err := responseBytesToWire(responseBytes, opts)
	if err != nil {
		return wrapSerializeException(err, "could not deserialize result")
	}

	if spec.ResultSpec == nil || spec.ResultSpec.ReturnType == nil {
		if len(w.Fields) == 0 {
			return nil
		}
		if w.Fields[0].ID == 0 {
			return fmt.Errorf("void method got unexpected result, fields: %+v", w.Fields)
		}
		return fmt.Errorf("void method got exception: %s", checkException(spec, w.Fields[0].ID))
	}

	if len(w.Fields) != 1 {
		return fmt.Errorf("method with return did not get 1 field in result: %+v", w.Fields)
	}

	if w.Fields[0].ID != 0 {
		return fmt.Errorf("method with return got exception: %s", checkException(spec, w.Fields[0].ID))
	}

	return nil
}

func responseBytesToWire(responseBytes []byte, opts Options) (wire.Struct, error) {
	var w wire.Value
	var err error

	reader := bytes.NewReader(responseBytes)
	if opts.UseEnvelopes {
		w, _, err = envelope.ReadReply(protocol.Binary, bytes.NewReader(responseBytes))
		if err != nil {
			return wire.Struct{}, encodedException{err}
		}
	} else {
		w, err = protocol.Binary.Decode(reader, wire.TStruct)
		if err != nil {
			return wire.Struct{}, fmt.Errorf("cannot parse Thrift struct from response: %v", err)
		}
	}

	if w.Type() != wire.TStruct {
		panic("Got unexpected type when parsing struct")
	}

	return w.GetStruct(), nil
}

func wrapSerializeException(err error, msg string) error {
	if _, ok := err.(encodedException); ok {
		return err
	}
	return fmt.Errorf("%s: %v", msg, err)
}
