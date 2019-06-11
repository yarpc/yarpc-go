// Copyright (c) 2019 Uber Technologies, Inc.
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
	"strings"

	"github.com/gogo/googleapis/google/rpc"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
	"go.uber.org/multierr"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
)

type pberror struct {
	code    yarpcerrors.Code
	message string
	details []proto.Message
}

func (err *pberror) Error() string {
	var b strings.Builder
	b.WriteString("code:")
	b.WriteString(err.code.String())
	if err.message != "" {
		b.WriteString(" message:")
		b.WriteString(err.message)
	}
	return b.String()
}

// NewError returns a new YARPC protobuf error. To access the error's fields, use the
// provided `GetError*` methods. These methods are nil-safe and
// provide default values for non-YARPC-errors.
//
// If the Code is CodeOK, this will return nil.
func NewError(code yarpcerrors.Code, message string, options ...ErrorOption) error {
	if code == yarpcerrors.CodeOK {
		return nil
	}
	pbErr := &pberror{
		code:    code,
		message: message,
	}
	for _, opt := range options {
		opt.apply(pbErr)
	}
	return pbErr
}

// GetErrorCode returns the error code of the error.
func GetErrorCode(err error) yarpcerrors.Code {
	if err == nil {
		return yarpcerrors.CodeOK
	}

	if pberr, ok := err.(*pberror); ok {
		return pberr.code
	}
	return yarpcerrors.CodeUnknown
}

// GetErrorMessage returns the error message of the error.
func GetErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	if pberr, ok := err.(*pberror); ok {
		return pberr.message
	}
	return err.Error()
}

// GetErrorDetails returns the error details of the error.
func GetErrorDetails(err error) []proto.Message {
	if err == nil {
		return nil
	}

	if pberr, ok := err.(*pberror); ok {
		return pberr.details
	}
	return nil
}

// ErrorOption is an option for the NewError constructor.
type ErrorOption struct{ apply func(*pberror) }

// WithErrorDetails sets the details of the error.
func WithErrorDetails(details ...proto.Message) ErrorOption {
	return ErrorOption{func(err *pberror) {
		err.details = append(err.details, details...)
	}}
}

func convertToYARPCError(encoding transport.Encoding, err error) error {
	if err == nil {
		return nil
	}
	if pberr, ok := err.(*pberror); ok {
		st, convertErr := status.New(_codeToGRPCCode[pberr.code], pberr.message).WithDetails(pberr.details...)
		if convertErr != nil {
			return convertErr
		}
		detailsBytes, cleanup, marshalErr := marshal(encoding, st.Proto())
		defer cleanup()
		if marshalErr != nil {
			return marshalErr
		}
		yarpcDet := make([]byte, len(detailsBytes))
		copy(yarpcDet, detailsBytes)
		return yarpcerrors.Newf(pberr.code, pberr.message).WithDetails(yarpcDet)
	}
	return err
}

func convertFromYARPCError(encoding transport.Encoding, err error) error {
	if err == nil || !yarpcerrors.IsStatus(err) {
		return err
	}
	yarpcErr := yarpcerrors.FromError(err)
	if yarpcErr.Details() == nil {
		return err
	}
	st := &rpc.Status{}
	unmarshalErr := unmarshalBytes(encoding, yarpcErr.Details(), st)
	if unmarshalErr != nil {
		return unmarshalErr
	}

	details := status.FromProto(st).Details()
	var protobufDetails []proto.Message
	var detailsErrs []error
	for _, detail := range details {
		if protoMessage, ok := detail.(proto.Message); ok {
			protobufDetails = append(protobufDetails, protoMessage)
		}
		if detailsErr, ok := detail.(error); ok {
			detailsErrs = append(detailsErrs, detailsErr)
		}
	}
	if len(detailsErrs) != 0 {
		return multierr.Combine(detailsErrs...)
	}

	return NewError(yarpcErr.Code(), yarpcErr.Message(), WithErrorDetails(protobufDetails...))
}

func (err *pberror) YARPCError() *yarpcerrors.Status {
	if err == nil {
		return nil
	}
	return yarpcerrors.Newf(err.code, err.message)
}
