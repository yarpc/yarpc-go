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

package json

import (
	"encoding/json"
	"reflect"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/internal/encoding"
	"github.com/yarpc/yarpc-go/transport"

	"golang.org/x/net/context"
)

// jsonHandler adapts a user-provided high-level handler into a transport-level
// Handler.
//
// The wrapped function must already be in the correct format:
//
// 	f(reqMeta *json.ReqMeta, body $reqBody) ($resBody, *json.ResMeta, error)
type jsonHandler struct {
	reader  requestReader
	handler reflect.Value
}

func (h jsonHandler) Handle(ctx context.Context, treq *transport.Request, rw transport.ResponseWriter) error {
	treq.Encoding = Encoding
	// TODO(abg): Should we fail requests if Rpc-Encoding does not match?

	reqBody, err := h.reader.Read(json.NewDecoder(treq.Body))
	if err != nil {
		return encoding.RequestBodyDecodeError(treq, err)
	}

	reqMeta := ReqMeta{
		Context:   ctx,
		Procedure: treq.Procedure,
		Headers:   yarpc.Headers(treq.Headers),
	}

	results := h.handler.Call([]reflect.Value{reflect.ValueOf(&reqMeta), reqBody})

	if err := results[2].Interface(); err != nil {
		// TODO proper error types
		return err.(error)
	}

	if resMeta := results[1].Interface().(*ResMeta); resMeta != nil {
		rw.AddHeaders(transport.Headers(resMeta.Headers))
	}

	result := results[0].Interface()
	if err := json.NewEncoder(rw).Encode(result); err != nil {
		return encoding.ResponseBodyEncodeError(treq, err)
	}

	return nil
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
	err := d.Decode(value.Interface())
	return value, err
}

type mapReader struct {
	Type reflect.Type // Type of the map
}

func (r mapReader) Read(d *json.Decoder) (reflect.Value, error) {
	value := reflect.New(r.Type)
	err := d.Decode(value.Interface())
	return value.Elem(), err
}

type ifaceEmptyReader struct{}

func (ifaceEmptyReader) Read(d *json.Decoder) (reflect.Value, error) {
	value := reflect.New(_interfaceEmptyType)
	err := d.Decode(value.Interface())
	return value.Elem(), err
}
