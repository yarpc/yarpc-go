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

package yarpcerror

// WrapHandlerError is a convenience function to help wrap errors returned
// from a handler.
//
// If err is nil, WrapHandlerError returns nil.
// If err is a YARPC error, WrapHandlerError returns err with no changes.
// If err is not a YARPC error, WrapHandlerError returns a new YARPC error
// with code CodeUnknown and message err.Error(), along with
// service and procedure information.
func WrapHandlerError(err error, service string, procedure string) error {
	if err == nil {
		return nil
	}
	if yarpcErr, ok := err.(*encodingError); ok {
		return yarpcErr
	}
	return New(CodeUnknown, sprintf("error for service %q and procedure %q: %s", service, procedure, err.Error()))
}
