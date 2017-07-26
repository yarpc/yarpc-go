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
	"encoding/json"
	"fmt"

	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/wire"
)

// getFieldMap gets a map from the fieldID to the fieldSpec for a given FieldGroup.
func getFieldMap(fields compile.FieldGroup) map[int16]*compile.FieldSpec {
	specs := make(map[int16]*compile.FieldSpec)
	for _, f := range fields {
		specs[f.ID] = f
	}
	return specs
}

func FromWireValueStruct(spec *compile.StructSpec, w wire.Struct) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	specs := getFieldMap(spec.Fields)
	for _, f := range w.Fields {
		fSpec, ok := specs[f.ID]
		if !ok {
			// TODO: Return unknown fields in the map possibly with a key :unknown_field_[id]
			continue
		}

		var err error
		result[fSpec.Name], err = FromWireValue(fSpec.Type, f.Value)
		if err != nil {
			return nil, specStructFieldMismatch{fSpec.Name, err}
		}
	}

	for _, fSpec := range specs {
		if _, ok := result[fSpec.Name]; ok {
			continue
		}

		// TODO: Return a warning when there's required fields that are missing.
		// if fSpec.Required {
		// }

		if fSpec.Default != nil {
			result[fSpec.Name] = constToRequest(fSpec.Default)
		}
	}

	return result, nil
}

func FromWireValueList(spec *compile.ListSpec, w wire.ValueList) ([]interface{}, error) {
	result := make([]interface{}, w.Size())
	values := wire.ValueListToSlice(w)
	for i, v := range values {
		var err error
		result[i], err = FromWireValue(spec.ValueSpec, v)
		if err != nil {
			return nil, specListItemMismatch{i, err}
		}
	}
	return result, nil
}

func FromWireValueSet(spec *compile.SetSpec, w wire.ValueList) ([]interface{}, error) {
	// Since wire.Set and wire.List are exactly the same type, we can cast one to the other.
	return FromWireValueList(&compile.ListSpec{
		ValueSpec: spec.ValueSpec,
	}, w)
}

func FromWireValueMap(spec *compile.MapSpec, w wire.MapItemList) (map[string]interface{}, error) {
	result := make(map[string]interface{}, w.Size())
	values := wire.MapItemListToSlice(w)
	for _, v := range values {
		key, err := FromWireValue(spec.KeySpec, v.Key)
		if err != nil {
			return nil, specMapItemMismatch{"key", err}
		}

		value, err := FromWireValue(spec.ValueSpec, v.Value)
		if err != nil {
			return nil, specMapItemMismatch{"value", err}
		}

		// Note: If the key is not hashable, we marshal it to a string and use that.
		// We might want to use a []MapItem instead to represent the map.
		if keyS, ok := key.(string); ok {
			result[keyS] = value
			continue
		}

		bs, err := json.Marshal(key)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal key object: %v\nkey: %v", err, key)
		}

		result[string(bs)] = value
	}
	return result, nil
}

func mapEnumValueToName(enumSpec *compile.EnumSpec, result int32) interface{} {
	for _, item := range enumSpec.Items {
		if item.Value == result {
			return item.Name
		}
	}
	return fmt.Sprintf("%v(%v)", enumSpec.Name, result)
}

// FromWireValue converts the wire.Value to the specific type it represents.
func FromWireValue(spec compile.TypeSpec, w wire.Value) (interface{}, error) {
	if spec.TypeCode() != w.Type() {
		return nil, specTypeMismatch{specified: spec.TypeCode(), got: w.Type()}
	}

	var result interface{}
	var err error

	spec = compile.RootTypeSpec(spec)
	switch spec.TypeCode() {
	case wire.TBool:
		result = w.GetBool()
	case wire.TI8:
		result = w.GetI8()
	case wire.TI16:
		result = w.GetI16()
	case wire.TI32:
		if enumSpec, ok := spec.(*compile.EnumSpec); ok {
			result = mapEnumValueToName(enumSpec, w.GetI32())
		} else {
			result = w.GetI32()
		}
	case wire.TI64:
		result = w.GetI64()
	case wire.TDouble:
		result = w.GetDouble()
	case wire.TBinary:
		// Binary could be a string, or actual []byte
		if _, ok := spec.(*compile.StringSpec); ok {
			result = w.GetString()
		} else {
			result = w.GetBinary()
		}
	case wire.TStruct:
		result, err = FromWireValueStruct(spec.(*compile.StructSpec), w.GetStruct())
	case wire.TList:
		result, err = FromWireValueList(spec.(*compile.ListSpec), w.GetList())
	case wire.TSet:
		result, err = FromWireValueSet(spec.(*compile.SetSpec), w.GetSet())
	case wire.TMap:
		result, err = FromWireValueMap(spec.(*compile.MapSpec), w.GetMap())
	default:
		panic(fmt.Sprintf("FromWireValue got an unknown type: %v", spec))
	}

	if err != nil {
		return nil, specValueMismatch{spec.ThriftName(), err}
	}
	return result, nil
}
