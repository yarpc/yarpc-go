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

// Package thrift contains functionality for converting generic data structures
// to and from Thrift payloads.
package thrift

import (
	"bytes"
	"fmt"
	"strings"

	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/protocol"
	"go.uber.org/thriftrw/wire"
)

// Parse parses the given Thrift file.
func Parse(file string) (*compile.Module, error) {
	module, err := compile.Compile(file, compile.NonStrict())
	// thriftrw wraps errors, so we can't use os.IsNotExist here.
	if err != nil {
		// The user may have left off the ".thrift", so try appending .thrift
		if appendedModule, err2 := compile.Compile(file+".thrift", compile.NonStrict()); err2 == nil {
			module = appendedModule
			err = nil
		}
	}
	return module, err
}

// SplitMethod takes a method name like Service::Method and splits it
// into Service and Method.
func SplitMethod(fullMethod string) (svc, method string, err error) {
	parts := strings.Split(fullMethod, "::")
	switch len(parts) {
	case 1:
		return parts[0], "", nil
	case 2:
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("invalid Thrift method %q, expected Service::Method", fullMethod)
	}
}

// RequestToBytes takes a user request and converts it to the Thrift binary payload.
// It uses the method spec to convert the user request.
func RequestToBytes(method *compile.FunctionSpec, request map[string]interface{}, opts Options) ([]byte, error) {
	w, err := structToValue(compile.FieldGroup(method.ArgsSpec), request)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	if opts.UseEnvelopes {
		// Sequence IDs are unused, so use the default, 0.
		enveloped := wire.Envelope{
			Name:  opts.EnvelopeMethodPrefix + method.Name,
			Type:  wire.Call,
			Value: wire.NewValueStruct(w),
		}
		err = protocol.Binary.EncodeEnveloped(enveloped, buf)
	} else {
		err = protocol.Binary.Encode(wire.NewValueStruct(w), buf)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to convert Thrift value to bytes: %v", err)
	}

	return buf.Bytes(), nil
}
