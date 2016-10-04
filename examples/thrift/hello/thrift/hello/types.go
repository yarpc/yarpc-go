// Code generated by thriftrw
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

package hello

import (
	"errors"
	"fmt"
	"go.uber.org/thriftrw/wire"
	"strings"
)

type EchoRequest struct {
	Message string `json:"message"`
	Count   int16  `json:"count"`
}

func (v *EchoRequest) ToWire() (wire.Value, error) {
	var (
		fields [2]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	w, err = wire.NewValueString(v.Message), error(nil)
	if err != nil {
		return w, err
	}
	fields[i] = wire.Field{ID: 1, Value: w}
	i++
	w, err = wire.NewValueI16(v.Count), error(nil)
	if err != nil {
		return w, err
	}
	fields[i] = wire.Field{ID: 2, Value: w}
	i++
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *EchoRequest) FromWire(w wire.Value) error {
	var err error
	messageIsSet := false
	countIsSet := false
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TBinary {
				v.Message, err = field.Value.GetString(), error(nil)
				if err != nil {
					return err
				}
				messageIsSet = true
			}
		case 2:
			if field.Value.Type() == wire.TI16 {
				v.Count, err = field.Value.GetI16(), error(nil)
				if err != nil {
					return err
				}
				countIsSet = true
			}
		}
	}
	if !messageIsSet {
		return errors.New("field Message of EchoRequest is required")
	}
	if !countIsSet {
		return errors.New("field Count of EchoRequest is required")
	}
	return nil
}

func (v *EchoRequest) String() string {
	var fields [2]string
	i := 0
	fields[i] = fmt.Sprintf("Message: %v", v.Message)
	i++
	fields[i] = fmt.Sprintf("Count: %v", v.Count)
	i++
	return fmt.Sprintf("EchoRequest{%v}", strings.Join(fields[:i], ", "))
}

type EchoResponse struct {
	Message string `json:"message"`
	Count   int16  `json:"count"`
}

func (v *EchoResponse) ToWire() (wire.Value, error) {
	var (
		fields [2]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	w, err = wire.NewValueString(v.Message), error(nil)
	if err != nil {
		return w, err
	}
	fields[i] = wire.Field{ID: 1, Value: w}
	i++
	w, err = wire.NewValueI16(v.Count), error(nil)
	if err != nil {
		return w, err
	}
	fields[i] = wire.Field{ID: 2, Value: w}
	i++
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *EchoResponse) FromWire(w wire.Value) error {
	var err error
	messageIsSet := false
	countIsSet := false
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TBinary {
				v.Message, err = field.Value.GetString(), error(nil)
				if err != nil {
					return err
				}
				messageIsSet = true
			}
		case 2:
			if field.Value.Type() == wire.TI16 {
				v.Count, err = field.Value.GetI16(), error(nil)
				if err != nil {
					return err
				}
				countIsSet = true
			}
		}
	}
	if !messageIsSet {
		return errors.New("field Message of EchoResponse is required")
	}
	if !countIsSet {
		return errors.New("field Count of EchoResponse is required")
	}
	return nil
}

func (v *EchoResponse) String() string {
	var fields [2]string
	i := 0
	fields[i] = fmt.Sprintf("Message: %v", v.Message)
	i++
	fields[i] = fmt.Sprintf("Count: %v", v.Count)
	i++
	return fmt.Sprintf("EchoResponse{%v}", strings.Join(fields[:i], ", "))
}
