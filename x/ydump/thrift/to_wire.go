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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/thriftrw/ast"
	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/wire"
	"gopkg.in/yaml.v2"
)

var (
	errUsingSingleField   = errors.New("union value must only have a single value")
	errStructUseMapString = errors.New("struct must be specified using map[string]*")
	errMapUnknownType     = errors.New("map must be specified as a mapping")
)

func structValueMap(value interface{}) (map[string]interface{}, bool) {
	if m, ok := value.(map[string]interface{}); ok {
		return m, true
	}

	m, ok := value.(map[interface{}]interface{})
	if !ok {
		return nil, false
	}

	result := make(map[string]interface{})
	for k, v := range m {
		keyStr, ok := k.(string)
		if !ok {
			return nil, false
		}
		result[keyStr] = v
	}
	return result, true
}

func structToValue(fieldGroup compile.FieldGroup, value interface{}) (wire.Struct, error) {
	mapValue, ok := structValueMap(value)
	if !ok {
		return wire.Struct{}, errStructUseMapString
	}

	fields, err := fieldGroupToValue(fieldGroup, mapValue)
	if err != nil {
		return wire.Struct{}, err
	}

	return wire.Struct{Fields: fields}, nil
}

func listToValue(t string, spec compile.TypeSpec, value interface{}) (wire.ValueList, error) {
	valueList, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%v must be specified using list[*]", t)
	}

	values := make([]wire.Value, len(valueList))
	for i, v := range valueList {
		wv, err := ToWireValue(spec, v)
		if err != nil {
			return nil, fmt.Errorf("%v item failed: %v", t, err)
		}

		values[i] = wv
	}

	return wire.ValueListFromSlice(spec.TypeCode(), values), nil
}

func convertStringMap(m map[string]interface{}) map[interface{}]interface{} {
	res := make(map[interface{}]interface{})
	for k, v := range m {
		res[k] = v
	}
	return res
}

// We want to support JSON, which doesn't allow non-string keys.
// So if we receive a string key and the underlying Thrift type is not a string
// then we try to unmarshal the JSON within the string.
func convertMapKey(keySpec compile.TypeSpec, value interface{}) interface{} {
	vStr, ok := value.(string)
	if !ok {
		// The user isn't nesting values inside of a string, ignore.
		return value
	}
	if keySpec.TypeCode() == wire.TBinary {
		return value
	}

	var data interface{}
	if err := yaml.Unmarshal([]byte(vStr), &data); err == nil {
		return data
	}

	// Since we couldn't unmarshal the string as JSON, just use it directly.
	return value
}

// mapToValue converts a map from JSON to a wire.Map.
// TODO: Allow specifying maps using a []MapItem form so the user
// can cleanly use non-string/int keys.
func mapToValue(keySpec, valueSpec compile.TypeSpec, value interface{}) (wire.MapItemList, error) {
	var valueMap map[interface{}]interface{}
	if vm, ok := value.(map[interface{}]interface{}); ok {
		valueMap = vm
	} else if vm, ok := value.(map[string]interface{}); ok {
		valueMap = convertStringMap(vm)
	} else {
		return nil, errMapUnknownType
	}

	items := make([]wire.MapItem, 0, len(valueMap))
	for k, v := range valueMap {
		keyValue := convertMapKey(keySpec, k)
		kw, err := ToWireValue(keySpec, keyValue)
		if err != nil {
			return nil, fmt.Errorf("map key (%v) failed: %v", k, err)
		}

		vw, err := ToWireValue(valueSpec, v)
		if err != nil {
			return nil, fmt.Errorf("map value (%v) for key (%v) failed: %v", v, k, err)
		}

		items = append(items, wire.MapItem{
			Key:   kw,
			Value: vw,
		})
	}

	return wire.MapItemListFromSlice(
		keySpec.TypeCode(),
		valueSpec.TypeCode(),
		items,
	), nil
}

// checkStructValue checks that the given wire.Struct is valid for the given spec.
func checkStructValue(spec *compile.StructSpec, value wire.Struct) error {
	if spec.Type != ast.UnionType {
		return nil
	}

	if len(value.Fields) != 1 {
		return errUsingSingleField
	}

	return nil
}

func parseEnumString(spec *compile.EnumSpec, value string) (int64, error) {
	for _, item := range spec.Items {
		if strings.EqualFold(item.Name, value) {
			return int64(item.Value), nil
		}
	}

	// Try to parse Name(%d) format, which is how we format unknown enums.
	if strings.HasPrefix(value, spec.Name+"(") && strings.HasSuffix(value, ")") {
		num := value[len(spec.Name)+1 : len(value)-1]
		if v, err := strconv.ParseInt(num, 10, 32); err == nil {
			return v, nil
		}
	}

	return 0, fmt.Errorf("unrecognized enum %q, expected one of %v", value, spec.Items)
}

func parseEnum(spec *compile.EnumSpec, value interface{}) (int64, error) {
	// If the value is a string, then we need to check the enum mapping.
	if v, ok := value.(string); ok {
		return parseEnumString(spec, v)
	}

	// Otherwise, try to parse the enum as a number.
	return parseInt(value, 32)
}

// ToWireValue converts a dynamic request object (usually a map) to Thrift.
func ToWireValue(spec compile.TypeSpec, value interface{}) (w wire.Value, err error) {
	spec = compile.RootTypeSpec(spec)
	switch spec.TypeCode() {
	case wire.TBool:
		var boolValue bool
		boolValue, err = parseBool(value)
		w = wire.NewValueBool(boolValue)
	case wire.TI8:
		var intValue int64
		intValue, err = parseInt(value, 8)
		w = wire.NewValueI8(int8(intValue))
	case wire.TI16:
		var intValue int64
		intValue, err = parseInt(value, 16)
		w = wire.NewValueI16(int16(intValue))
	case wire.TI32:
		var intValue int64
		if enumSpec, ok := spec.(*compile.EnumSpec); ok {
			intValue, err = parseEnum(enumSpec, value)
		} else {
			intValue, err = parseInt(value, 32)
		}
		w = wire.NewValueI32(int32(intValue))
	case wire.TI64:
		var intValue int64
		intValue, err = parseInt(value, 64)
		w = wire.NewValueI64(int64(intValue))
	case wire.TDouble:
		var doubleValue float64
		doubleValue, err = parseDouble(value)
		w = wire.NewValueDouble(doubleValue)
	case wire.TBinary:
		var binaryValue []byte
		binaryValue, err = parseBinary(value)
		w = wire.NewValueBinary(binaryValue)
	case wire.TStruct:
		sspec := spec.(*compile.StructSpec)
		var structValue wire.Struct
		structValue, err = structToValue(sspec.Fields, value)
		if err == nil {
			err = checkStructValue(sspec, structValue)
		}
		w = wire.NewValueStruct(structValue)
	case wire.TList:
		lspec := spec.(*compile.ListSpec)
		var wireValue wire.ValueList
		wireValue, err = listToValue("list", lspec.ValueSpec, value)
		w = wire.NewValueList(wireValue)
	case wire.TSet:
		lspec := spec.(*compile.SetSpec)
		var wireValue wire.ValueList
		wireValue, err = listToValue("set", lspec.ValueSpec, value)
		w = wire.NewValueSet(wireValue)
	case wire.TMap:
		mspec := spec.(*compile.MapSpec)
		var wireValue wire.MapItemList
		wireValue, err = mapToValue(mspec.KeySpec, mspec.ValueSpec, value)
		w = wire.NewValueMap(wireValue)
	default:
		panic(fmt.Sprintf("got unknown TypeCode in spec: %v", spec))
	}

	if err != nil {
		// TODO: Error messages should use the thrift field name, not type.
		return wire.Value{}, fmt.Errorf("field %q %v", spec.ThriftName(), err)
	}
	return w, nil
}
