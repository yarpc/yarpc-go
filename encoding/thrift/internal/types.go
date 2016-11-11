// Code generated by thriftrw v0.5.0
// @generated

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

package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/thriftrw/wire"
	"math"
	"strconv"
	"strings"
)

type ExceptionType int32

const (
	ExceptionTypeUnknown               ExceptionType = 0
	ExceptionTypeUnknownMethod         ExceptionType = 1
	ExceptionTypeInvalidMessageType    ExceptionType = 2
	ExceptionTypeWrongMethodName       ExceptionType = 3
	ExceptionTypeBadSequenceID         ExceptionType = 4
	ExceptionTypeMissingResult         ExceptionType = 5
	ExceptionTypeInternalError         ExceptionType = 6
	ExceptionTypeProtocolError         ExceptionType = 7
	ExceptionTypeInvalidTransform      ExceptionType = 8
	ExceptionTypeInvalidProtocol       ExceptionType = 9
	ExceptionTypeUnsupportedClientType ExceptionType = 10
)

func (v ExceptionType) ToWire() (wire.Value, error) {
	return wire.NewValueI32(int32(v)), nil
}

func (v *ExceptionType) FromWire(w wire.Value) error {
	*v = (ExceptionType)(w.GetI32())
	return nil
}

func (v ExceptionType) String() string {
	w := int32(v)
	switch w {
	case 0:
		return "UNKNOWN"
	case 1:
		return "UNKNOWN_METHOD"
	case 2:
		return "INVALID_MESSAGE_TYPE"
	case 3:
		return "WRONG_METHOD_NAME"
	case 4:
		return "BAD_SEQUENCE_ID"
	case 5:
		return "MISSING_RESULT"
	case 6:
		return "INTERNAL_ERROR"
	case 7:
		return "PROTOCOL_ERROR"
	case 8:
		return "INVALID_TRANSFORM"
	case 9:
		return "INVALID_PROTOCOL"
	case 10:
		return "UNSUPPORTED_CLIENT_TYPE"
	}
	return fmt.Sprintf("ExceptionType(%d)", w)
}

func (v ExceptionType) MarshalJSON() ([]byte, error) {
	switch int32(v) {
	case 0:
		return ([]byte)("\"UNKNOWN\""), nil
	case 1:
		return ([]byte)("\"UNKNOWN_METHOD\""), nil
	case 2:
		return ([]byte)("\"INVALID_MESSAGE_TYPE\""), nil
	case 3:
		return ([]byte)("\"WRONG_METHOD_NAME\""), nil
	case 4:
		return ([]byte)("\"BAD_SEQUENCE_ID\""), nil
	case 5:
		return ([]byte)("\"MISSING_RESULT\""), nil
	case 6:
		return ([]byte)("\"INTERNAL_ERROR\""), nil
	case 7:
		return ([]byte)("\"PROTOCOL_ERROR\""), nil
	case 8:
		return ([]byte)("\"INVALID_TRANSFORM\""), nil
	case 9:
		return ([]byte)("\"INVALID_PROTOCOL\""), nil
	case 10:
		return ([]byte)("\"UNSUPPORTED_CLIENT_TYPE\""), nil
	}
	return ([]byte)(strconv.FormatInt(int64(v), 10)), nil
}

func (v *ExceptionType) UnmarshalJSON(text []byte) error {
	d := json.NewDecoder(bytes.NewReader(text))
	d.UseNumber()
	t, err := d.Token()
	if err != nil {
		return err
	}
	switch w := t.(type) {
	case json.Number:
		x, err := w.Int64()
		if err != nil {
			return err
		}
		if x > math.MaxInt32 {
			return fmt.Errorf("enum overflow from JSON %q for %q", text, "ExceptionType")
		}
		if x < math.MinInt32 {
			return fmt.Errorf("enum underflow from JSON %q for %q", text, "ExceptionType")
		}
		*v = (ExceptionType)(x)
		return nil
	case string:
		switch w {
		case "UNKNOWN":
			*v = ExceptionTypeUnknown
			return nil
		case "UNKNOWN_METHOD":
			*v = ExceptionTypeUnknownMethod
			return nil
		case "INVALID_MESSAGE_TYPE":
			*v = ExceptionTypeInvalidMessageType
			return nil
		case "WRONG_METHOD_NAME":
			*v = ExceptionTypeWrongMethodName
			return nil
		case "BAD_SEQUENCE_ID":
			*v = ExceptionTypeBadSequenceID
			return nil
		case "MISSING_RESULT":
			*v = ExceptionTypeMissingResult
			return nil
		case "INTERNAL_ERROR":
			*v = ExceptionTypeInternalError
			return nil
		case "PROTOCOL_ERROR":
			*v = ExceptionTypeProtocolError
			return nil
		case "INVALID_TRANSFORM":
			*v = ExceptionTypeInvalidTransform
			return nil
		case "INVALID_PROTOCOL":
			*v = ExceptionTypeInvalidProtocol
			return nil
		case "UNSUPPORTED_CLIENT_TYPE":
			*v = ExceptionTypeUnsupportedClientType
			return nil
		default:
			return fmt.Errorf("unknown enum value %q for %q", w, "ExceptionType")
		}
	default:
		return fmt.Errorf("invalid JSON value %q (%T) to unmarshal into %q", t, t, "ExceptionType")
	}
}

type TApplicationException struct {
	Message *string        `json:"message,omitempty"`
	Type    *ExceptionType `json:"type,omitempty"`
}

func (v *TApplicationException) ToWire() (wire.Value, error) {
	var (
		fields [2]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Message != nil {
		w, err = wire.NewValueString(*(v.Message)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 1, Value: w}
		i++
	}
	if v.Type != nil {
		w, err = v.Type.ToWire()
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 2, Value: w}
		i++
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func _ExceptionType_Read(w wire.Value) (ExceptionType, error) {
	var v ExceptionType
	err := v.FromWire(w)
	return v, err
}

func (v *TApplicationException) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TBinary {
				var x string
				x, err = field.Value.GetString(), error(nil)
				v.Message = &x
				if err != nil {
					return err
				}
			}
		case 2:
			if field.Value.Type() == wire.TI32 {
				var x ExceptionType
				x, err = _ExceptionType_Read(field.Value)
				v.Type = &x
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *TApplicationException) String() string {
	var fields [2]string
	i := 0
	if v.Message != nil {
		fields[i] = fmt.Sprintf("Message: %v", *(v.Message))
		i++
	}
	if v.Type != nil {
		fields[i] = fmt.Sprintf("Type: %v", *(v.Type))
		i++
	}
	return fmt.Sprintf("TApplicationException{%v}", strings.Join(fields[:i], ", "))
}

func (v *TApplicationException) Error() string {
	return v.String()
}
