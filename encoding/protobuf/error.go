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
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/grpcerrorcodes"
	"go.uber.org/yarpc/yarpcerrors"
)

type pberror struct {
	code    yarpcerrors.Code
	message string
	details []interface{}
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

// NewError returns a new YARPC protobuf error. To access the error's fields,
// use the yarpcerrors package APIs for the code and message, and the
// `GetErrorDetails(error)` function for error details. The `yarpcerrors.Details()`
// will not work on this error.
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

// GetErrorDetails returns the error details of the error.
//
// Each element in the returned slice of interface{} is either a proto.Message
// or an error to explain why the element is not a proto.Message, most likely
// because the error detail could not be unmarshaled.
// See: https://github.com/gogo/status/blob/master/status.go#L193
func GetErrorDetails(err error) []interface{} {
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

// WithErrorDetails adds to the details of the error.
func WithErrorDetails(details ...proto.Message) ErrorOption {
	return ErrorOption{func(err *pberror) {
		for _, detail := range details {
			err.details = append(err.details, detail)
		}
	}}
}

// convertToYARPCError is to be used for handling errors on the inbound side.
func convertToYARPCError(encoding transport.Encoding, err error) error {
	if err == nil {
		return nil
	}
	if pberr, ok := err.(*pberror); ok {
		// We only use this function on the inbound side, and pberrors should be
		// constructed using the constructor above, so we can safely assume all
		// the details are proto.Message-typed.
		var details []proto.Message
		for _, detail := range pberr.details {
			details = append(details, detail.(proto.Message))
		}
		st, convertErr := status.New(grpcerrorcodes.YARPCCodeToGRPCCode[pberr.code], pberr.message).WithDetails(details...)
		if convertErr != nil {
			return convertErr
		}
		detailsBytes, cleanup, marshalErr := marshal(encoding, st.Proto())
		if marshalErr != nil {
			return marshalErr
		}
		defer cleanup()
		yarpcDet := make([]byte, len(detailsBytes))
		copy(yarpcDet, detailsBytes)
		return yarpcerrors.Newf(pberr.code, pberr.message).WithDetails(yarpcDet)
	}
	return err
}

// convertFromYARPCError is to be used for handling errors on the outbound side.
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
	return newErrorWithDetails(yarpcErr.Code(), yarpcErr.Message(), details)
}

func newErrorWithDetails(code yarpcerrors.Code, message string, details []interface{}) error {
	return &pberror{
		code:    code,
		message: message,
		details: details,
	}
}

func (err *pberror) YARPCError() *yarpcerrors.Status {
	if err == nil {
		return nil
	}
	return yarpcerrors.Newf(err.code, err.message)
}
