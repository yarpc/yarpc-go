// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpcjson

import (
	"encoding/json"
	"reflect"

	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
)

var (
	_ requestReader = structReader{}
	_ requestReader = ifaceEmptyReader{}
	_ requestReader = mapReader{}
)

type jsonCodec struct {
	reader requestReader
}

// newCodec constructs a JSON codec. The handler's signature should be verified before newCodec is called.
func newCodec(handler interface{}) jsonCodec {
	reqBodyType := reflect.TypeOf(handler).In(1)
	var r requestReader
	if reqBodyType == _interfaceEmptyType {
		r = ifaceEmptyReader{}
	} else if reqBodyType.Kind() == reflect.Map {
		r = mapReader{reqBodyType}
	} else {
		r = structReader{reqBodyType.Elem()}
	}

	return jsonCodec{
		reader: r,
	}
}

func (c jsonCodec) Decode(res *yarpc.Buffer) (interface{}, error) {
	reqBody, err := c.reader.Read(json.NewDecoder(res))
	if err != nil {
		return nil, err
	}

	return reqBody.Interface(), nil
}

func (c jsonCodec) Encode(res interface{}) (*yarpc.Buffer, error) {
	resBuf := &yarpc.Buffer{}
	if err := json.NewEncoder(resBuf).Encode(res); err != nil {
		return nil, err
	}

	return resBuf, nil
}

func (c jsonCodec) EncodeError(err error) (*yarpc.Buffer, error) {
	details := yarpcerror.GetDetails(err)
	if details == nil {
		return nil, nil
	}
	resBuf := &yarpc.Buffer{}
	if err := json.NewEncoder(resBuf).Encode(details); err != nil {
		return nil, err
	}

	return resBuf, nil
}

// requestReader is used to parse a JSON request argument from a JSON decoder.
type requestReader interface {
	Read(*json.Decoder) (reflect.Value, error)
}

type structReader struct {
	// Type of the struct (not a pointer to the struct)
	Type reflect.Type
}

func (r structReader) Read(d *json.Decoder) (reflect.Value, error) {
	value := reflect.New(r.Type)
	if err := d.Decode(value.Interface()); err != nil {
		return reflect.Value{}, err
	}

	return value, nil
}

type mapReader struct {
	// Type of the map
	Type reflect.Type
}

func (r mapReader) Read(d *json.Decoder) (reflect.Value, error) {
	value := reflect.New(r.Type)
	if err := d.Decode(value.Interface()); err != nil {
		return reflect.Value{}, err
	}

	return value.Elem(), nil
}

type ifaceEmptyReader struct{}

func (ifaceEmptyReader) Read(d *json.Decoder) (reflect.Value, error) {
	value := reflect.New(_interfaceEmptyType)
	if err := d.Decode(value.Interface()); err != nil {
		return reflect.Value{}, err
	}

	return value.Elem(), nil
}
