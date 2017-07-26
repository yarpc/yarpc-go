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
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

var errBinaryObjectOptions = errors.New(
	"object input for binary/string must have one of the following keys: base64, file")

func parseBoolNumber(v int) (bool, error) {
	switch v {
	case 0:
		return false, nil
	case 1:
		return true, nil
	}

	return false, fmt.Errorf("cannot parse bool from int %v", v)
}

// parseBool parses a boolean from a bool, string or a number.
// If a string is given, it must be "true" or "false" (case insensitive)
// If a number is given, it must be 1 for true, or 0 for false.
func parseBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case int:
		// YAML uses int for all small values. For large values, it may use int64 or uint64
		// but those are not valid bool values anyway, so we don't need to check them.
		return parseBoolNumber(v)
	case string:
		return strconv.ParseBool(strings.ToLower(v))
	default:
		return false, fmt.Errorf("cannot parse bool %T: %v", value, value)
	}
}

// parseInt parses an integer of the given size.
func parseInt(v interface{}, bits int) (int64, error) {
	var maxVal int64 = 1<<(uint(bits)-1) - 1
	minVal := -maxVal - 1

	var v64 int64
	switch v := v.(type) {
	case int64:
		v64 = v
	case int:
		v64 = int64(v)
	case int8:
		v64 = int64(v)
	case int16:
		v64 = int64(v)
	case int32:
		v64 = int64(v)
	case uint64:
		// YAML will only use uint64 if the value doesn't fit in an int64.
		// However, Thrift only supports int64 values.
		return 0, fmt.Errorf("uint64 value %v is out of range for int%v [%v, %v]", v, bits, minVal, maxVal)
	default:
		return 0, fmt.Errorf("cannot parse int%v from %T: %v", bits, v, v)
	}

	if v64 < minVal || v64 > maxVal {
		return 0, fmt.Errorf("value %v is out of range for int%v [%v, %v]", v64, bits, minVal, maxVal)
	}

	return v64, nil
}

// parseDouble parses a float64 from an integer or a float.
func parseDouble(v interface{}) (float64, error) {
	switch v := v.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("cannot parse double from %T: %v", v, v)
	}
}

// parseBinaryList will try to parse a list of numbers
// or a list of strings.
func parseBinaryList(vl []interface{}) ([]byte, error) {
	bs := make([]byte, 0, len(vl))
	for _, v := range vl {
		switch v := v.(type) {
		case int:
			// YAML uses int for all small values. For large values, it may use int64 or uint64
			// but those are not valid bool values anyway, so we don't need to check them.
			if v < 0 || v >= (1<<8) {
				return nil, fmt.Errorf("failed to parse list of bytes: %v is not a byte", v)
			}
			bs = append(bs, byte(v))
		case string:
			bs = append(bs, v...)
		default:
			return nil, fmt.Errorf("can only parse list of bytes or characters, invalid element: %q", v)
		}
	}

	return bs, nil
}

func parseBinaryMap(v map[interface{}]interface{}) ([]byte, error) {
	if v, ok := v["base64"]; ok {
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("base64 must be specified as string, got: %T", v)
		}

		// Since we don't know whether the user's input is padded or not, we strip
		// all "=" characters out, and use RawStdEncoding (which does not need padding).
		str = strings.TrimRight(str, "=")
		return base64.RawStdEncoding.DecodeString(str)
	}

	if v, ok := v["file"]; ok {
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("file requires filename as string, got %T", v)
		}

		return ioutil.ReadFile(str)
	}

	return nil, errBinaryObjectOptions
}

// parseBinary can parse a string or binary.
// If a string is given, it is used as the binary value directly.
func parseBinary(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	case []interface{}:
		return parseBinaryList(v)
	case map[interface{}]interface{}:
		return parseBinaryMap(v)
	case bool, int, int8, int16, int32, int64, uint64, float32, float64:
		// Since YAML tries to guess the type of a value, if the value
		// looks like an integer, YAML will use int types even though the
		// user needs a string. E.g., `msg: true` will treat "true" as a bool,
		// but `msg: t` will be treat "t"" as a string. So instead of throwing
		// an error at the user, coerce known YAML scalar types to a string.
		return []byte(fmt.Sprint(v)), nil
	default:
		return nil, fmt.Errorf("cannot parse binary/string from %T: %v", value, v)
	}
}
