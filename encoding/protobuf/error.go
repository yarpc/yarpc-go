// Copyright (c) 2020 Uber Technologies, Inc.
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
	"strings"

	// lint:file-ignore SA1019 no need to migrate to google.golang.org/protobuf yet
	v1proto "github.com/golang/protobuf/proto"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/grpcerrorcodes"
	"go.uber.org/yarpc/yarpcerrors"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
)

const (
	// format for converting error details to string
	_errDetailsFmt = "[]{ %s }"
	// format for converting a single message to string
	_errDetailFmt = "%s{%s}"
)

var _ error = (*pberror)(nil)

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
// This method supports extracting details from wrapped errors.
//
// Each element in the returned slice of interface{} is either a proto.Message
// or an error to explain why the element is not a proto.Message, most likely
// because the error detail could not be unmarshaled.
// See: https://godoc.org/google.golang.org/grpc/internal/status#Status.Details
func GetErrorDetails(err error) []interface{} {
	if err == nil {
		return nil
	}
	var target *pberror
	if errors.As(err, &target) {
		return target.details
	}
	return nil
}

// ErrorOption is an option for the NewError constructor.
type ErrorOption struct{ apply func(*pberror) }

// WithErrorDetails adds to the details of the error.
func WithErrorDetails(details ...v1proto.Message) ErrorOption {
	return ErrorOption{func(err *pberror) {
		for _, detail := range details {
			err.details = append(err.details, detail)
		}
	}}
}

// convertToYARPCError is to be used for handling errors on the inbound side.
func convertToYARPCError(encoding transport.Encoding, err error, codec *codec, resw transport.ResponseWriter) error {
	if err == nil {
		return nil
	}
	var pberr *pberror
	if errors.As(err, &pberr) {
		setApplicationErrorMeta(pberr, resw)
		status, sterr := createStatusWithDetail(pberr, encoding, codec)
		if sterr != nil {
			return sterr
		}
		return status
	}
	return err
}

func createStatusWithDetail(pberr *pberror, encoding transport.Encoding, codec *codec) (*yarpcerrors.Status, error) {
	details := make([]v1proto.Message, 0, len(pberr.details))
	for _, detail := range pberr.details {
		if pbdetail, ok := detail.(v1proto.Message); ok {
			details = append(details, pbdetail)
		} else {
			return nil, errors.New("proto error detail is not proto.Message compatible")
		}
	}

	st, convertErr := status.New(grpcerrorcodes.YARPCCodeToGRPCCode[pberr.code], pberr.message).WithDetails(details...)
	if convertErr != nil {
		return nil, convertErr
	}
	detailsBytes, marshalErr := marshal(encoding, st.Proto(), codec)
	if marshalErr != nil {
		return nil, marshalErr
	}
	yarpcDet := make([]byte, len(detailsBytes))
	copy(yarpcDet, detailsBytes)
	return yarpcerrors.Newf(pberr.code, pberr.message).WithDetails(yarpcDet), nil
}

func setApplicationErrorMeta(pberr *pberror, resw transport.ResponseWriter) {
	applicationErroMetaSetter, ok := resw.(transport.ApplicationErrorMetaSetter)
	if !ok {
		return
	}

	var appErrName string
	if len(pberr.details) > 0 { // only grab the first name since this will be emitted with metrics
		appErrName = messageNameWithoutPackage(v1proto.MessageName(
			pberr.details[0].(v1proto.Message)),
		)
	}

	details := make([]string, 0, len(pberr.details))
	for _, detail := range pberr.details {
		details = append(details, protobufMessageToString(detail.(v1proto.Message)))
	}

	applicationErroMetaSetter.SetApplicationErrorMeta(&transport.ApplicationErrorMeta{
		Name:    appErrName,
		Details: fmt.Sprintf(_errDetailsFmt, strings.Join(details, " , ")),
	})
}

// messageNameWithoutPackage strips the package name, returning just the type
// name.
//
// For example:
//  uber.foo.bar.TypeName -> TypeName
func messageNameWithoutPackage(messageName string) string {
	if i := strings.LastIndex(messageName, "."); i >= 0 {
		return messageName[i+1:]
	}
	return messageName
}

func protobufMessageToString(message v1proto.Message) string {
	return fmt.Sprintf(_errDetailFmt,
		messageNameWithoutPackage(v1proto.MessageName(message)),
		v1proto.CompactTextString(message))
}

// convertFromYARPCError is to be used for handling errors on the outbound side.
func convertFromYARPCError(encoding transport.Encoding, err error, codec *codec) error {
	if err == nil || !yarpcerrors.IsStatus(err) {
		return err
	}
	yarpcErr := yarpcerrors.FromError(err)
	if yarpcErr.Details() == nil {
		return err
	}
	st := &spb.Status{}
	unmarshalErr := unmarshalBytes(encoding, yarpcErr.Details(), v1proto.MessageV2(st), codec)
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
	status, statusErr := createStatusWithDetail(err, Encoding, newCodec(nil))
	if statusErr != nil {
		return yarpcerrors.FromError(statusErr)
	}
	return status
}
