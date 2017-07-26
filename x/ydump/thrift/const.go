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
	"fmt"

	"go.uber.org/thriftrw/compile"
)

func constToRequest(v compile.ConstantValue) interface{} {
	switch v := v.(type) {
	case compile.ConstantBool:
		return bool(v)
	case compile.ConstantDouble:
		return float64(v)
	case compile.ConstantInt:
		return int64(v)
	case compile.ConstantString:
		return string(v)
	case compile.ConstReference:
		return constToRequest(v.Target.Value)
	case compile.EnumItemReference:
		return v.Item.Value
	case compile.ConstantSet:
		return constValueListToRequest([]compile.ConstantValue(v))
	case compile.ConstantList:
		return constValueListToRequest([]compile.ConstantValue(v))
	case compile.ConstantMap:
		result := make(map[interface{}]interface{})
		for _, value := range v {
			// Note: If the key is not hashable, this will fail.
			// We should allow using a custom []MapItem to represent a map.
			result[constToRequest(value.Key)] = constToRequest(value.Value)
		}
		return result
	case *compile.ConstantStruct:
		result := make(map[string]interface{})
		for key, value := range v.Fields {
			result[key] = constToRequest(value)
		}
		return result
	default:
		panic(fmt.Sprintf("unknown constant type: %T", v))
	}
}

func constValueListToRequest(v []compile.ConstantValue) interface{} {
	result := make([]interface{}, len(v))
	for i, value := range v {
		result[i] = constToRequest(value)
	}
	return result
}
