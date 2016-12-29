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

package protobuf

import (
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
)

var (
	errNilApplicationErrorMessage = errors.New("a nil proto.Message was passed to protobuf.NewApplicationError")
)

// ApplicationError is an error that occured on the application side, if included as
// one of the error types marked on the method in the protobuf definition.
//
// This will be the returned error from a client if such an error occured.
//
// If the server wants to return such an error, it should do:
//
//   return nil, nil, protobuf.NewApplicationError(errorMessage)
type ApplicationError struct {
	message proto.Message
}

// NewApplicationError returns a new ApplicationError for server-side calls.
//
// If message is nil, this will return an error that specifies you passed a nil message.
func NewApplicationError(message proto.Message) error {
	if message == nil {
		return errNilApplicationErrorMessage
	}
	return &ApplicationError{message}
}

// Error implements the error interface.
func (a *ApplicationError) Error() string {
	return fmt.Sprintf("%T<%+v>", a.message, a.message)
}

// ClientResponseCastError returns an error saying the response type was not the expected type on the client-side.
//
// This should only be used in generated code.
func ClientResponseCastError(serviceName string, methodName string, expectedType proto.Message, actualType proto.Message) error {
	return castError("client", "response", serviceName, methodName, expectedType, actualType)
}

// ServerRequestCastError returns an error saying the request type was not the expected type on the server-side.
//
// This should only be used in generated code.
func ServerRequestCastError(serviceName string, methodName string, expectedType proto.Message, actualType proto.Message) error {
	return castError("server", "request", serviceName, methodName, expectedType, actualType)
}

func castError(
	clientOrServerString string,
	requestOrResponseString string,
	serviceName string,
	methodName string,
	expectedType proto.Message,
	actualType proto.Message,
) error {
	return fmt.Errorf(
		"%s.%s %s call was expected to have %s type %T but had type %T",
		serviceName,
		methodName,
		clientOrServerString,
		requestOrResponseString,
		expectedType,
		actualType,
	)
}
