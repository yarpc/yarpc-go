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

package v2

import (
	"errors"
	"fmt"
	"strings"

	golang_proto "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/grpcerrorcodes"
	"go.uber.org/yarpc/yarpcerrors"
	rpc_status "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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
	details []*any.Any
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
		if err := opt.apply(pbErr); err != nil {
			return err
		}
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
func GetErrorDetails(err error) []interface{} {
	var target *pberror
	if errors.As(err, &target) && len(target.details) > 0 {
		results := make([]interface{}, 0, len(target.details))
		for _, ghAny := range target.details {
			detail, err := ghAny.UnmarshalNew()
			if err != nil {
				results = append(results, err)
				continue
			}
			results = append(results, detail)
		}
		return results
	}
	return nil
}

// ErrorOption is an option for the NewError constructor.
type ErrorOption struct{ apply func(*pberror) error }

// WithErrorDetails adds to the details of the error.
// If any errors are encountered, it returns the first error encountered.
func WithErrorDetails(details ...proto.Message) ErrorOption {
	return ErrorOption{func(err *pberror) error {
		for _, detail := range details {
			any, terr := anypb.New(detail)
			if terr != nil {
				return terr
			}
			err.details = append(err.details, any)
		}
		return nil
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
	if pberr.code == yarpcerrors.CodeOK {
		return nil, errors.New("no status error for error with code OK")
	}

	st := status.New(grpcerrorcodes.YARPCCodeToGRPCCode[pberr.code], pberr.message)
	// Here we check that status.New has returned a valid error.
	if st.Err() == nil {
		return nil, fmt.Errorf("no status error for error with code %d", pberr.code)
	}
	pst := st.Proto()
	pst.Details = pberr.details

	detailsBytes, cleanup, marshalErr := marshal(encoding, golang_proto.MessageV2(pst), codec)
	if marshalErr != nil {
		return nil, marshalErr
	}
	defer cleanup()
	yarpcDet := make([]byte, len(detailsBytes))
	copy(yarpcDet, detailsBytes)
	return yarpcerrors.Newf(pberr.code, pberr.message).WithDetails(yarpcDet), nil
}

func setApplicationErrorMeta(pberr *pberror, resw transport.ResponseWriter) {
	applicationErroMetaSetter, ok := resw.(transport.ApplicationErrorMetaSetter)
	if !ok {
		return
	}

	decodedDetails := GetErrorDetails(pberr)
	var appErrName string
	if len(decodedDetails) > 0 { // only grab the first name since this will be emitted with metrics
		decodedMsg := decodedDetails[0].(proto.Message)
		appErrName = messageNameWithoutPackage(string(proto.MessageName(decodedMsg)))
	}

	details := make([]string, 0, len(decodedDetails))
	for _, detail := range decodedDetails {
		details = append(details, protobufMessageToString(detail.(proto.Message)))
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

func protobufMessageToString(message proto.Message) string {

	return fmt.Sprintf(_errDetailFmt,
		messageNameWithoutPackage(string(proto.MessageName(message))),
		golang_proto.CompactTextString(golang_proto.MessageV1(message)))
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
	st := &rpc_status.Status{}
	unmarshalErr := unmarshalBytes(encoding, yarpcErr.Details(), golang_proto.MessageV2(st), codec)
	if unmarshalErr != nil {
		return unmarshalErr
	}

	return newErrorWithDetails(yarpcErr.Code(), yarpcErr.Message(), st.GetDetails())
}

func newErrorWithDetails(code yarpcerrors.Code, message string, details []*any.Any) error {
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
