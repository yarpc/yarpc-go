// Copyright (c) 2017 Uber Technologies, Inc.
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

package transport

import "go.uber.org/zap/zapcore"

// RequestFeatures are features that the client implements.
//
// By setting a feature, the client signifies that it can handle certain
// features, and the server can choose how to proceed. If the feature is
// used, the server will return a corresponding signal on ResponseFeatures.
//
// This is needed for backwards compatibility.
type RequestFeatures struct {
	// AcceptsBothResponseError indicates that the client can handle both
	// a response body and error at the same time.
	AcceptsBothResponseError bool
}

// MarshalLogObject implements zap.ObjectMarshaler.
func (f RequestFeatures) MarshalLogObject(objectEncoder zapcore.ObjectEncoder) error {
	objectEncoder.AddBool("acceptsBothResponseError", f.AcceptsBothResponseError)
	return nil
}

// ResponseFeatures are features that were applied on the server.
//
// The server can only use features that were signaled from RequestFeatures.
// If the feature is used, the server must indicate so on ResponseFeatures.
//
// This is needed for backwards compatibility.
type ResponseFeatures struct {
	// BothResponseError indicates that the server potentially retuurned both a
	// response body and error at the same time.
	BothResponseError bool
}

// MarshalLogObject implements zap.ObjectMarshaler.
func (f ResponseFeatures) MarshalLogObject(objectEncoder zapcore.ObjectEncoder) error {
	objectEncoder.AddBool("bothResponseError", f.BothResponseError)
	return nil
}
