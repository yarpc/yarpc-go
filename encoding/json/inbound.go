// Copyright (c) 2022 Uber Technologies, Inc.
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
	"context"
	"encoding/json"
	"reflect"

	encodingapi "go.uber.org/yarpc/api/encoding"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/errors"
)

// jsonHandler adapts a user-provided high-level handler into a transport-level
// Handler.
//
// The wrapped function must already be in the correct format:
//
//	f(ctx context.Context, body $reqBody) ($resBody, error)
type jsonHandler struct {
	reader  requestReader
	handler reflect.Value
}

func (h jsonHandler) Handle(ctx context.Context, treq *transport.Request, rw transport.ResponseWriter) error {
	if err := errors.ExpectEncodings(treq, Encoding); err != nil {
		return err
	}

	ctx, call := encodingapi.NewInboundCall(ctx)
	if err := call.ReadFromRequest(treq); err != nil {
		return err
	}

	reqBody, err := h.reader.Read(json.NewDecoder(treq.Body))
	if err != nil {
		return errors.RequestBodyDecodeError(treq, err)
	}

	results := h.handler.Call([]reflect.Value{reflect.ValueOf(ctx), reqBody})

	if err := call.WriteToResponse(rw); err != nil {
		return err
	}

	// we want to return the appErr if it exists as this is what
	// the previous behavior was so we deprioritize this error
	var encodeErr error
	if result := results[0].Interface(); result != nil {
		if err := json.NewEncoder(rw).Encode(result); err != nil {
			encodeErr = errors.ResponseBodyEncodeError(treq, err)
		}
	}

	if appErr, _ := results[1].Interface().(error); appErr != nil {
		rw.SetApplicationError()
		return appErr
	}

	return encodeErr
}

func (h jsonHandler) HandleOneway(ctx context.Context, treq *transport.Request) error {
	if err := errors.ExpectEncodings(treq, Encoding); err != nil {
		return err
	}

	ctx, call := encodingapi.NewInboundCall(ctx)
	if err := call.ReadFromRequest(treq); err != nil {
		return err
	}

	reqBody, err := h.reader.Read(json.NewDecoder(treq.Body))
	if err != nil {
		return errors.RequestBodyDecodeError(treq, err)
	}

	results := h.handler.Call([]reflect.Value{reflect.ValueOf(ctx), reqBody})

	if err := results[0].Interface(); err != nil {
		return err.(error)
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
