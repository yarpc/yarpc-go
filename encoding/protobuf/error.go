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

// GetErrorCode returns the error message of the error.
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

func IsProtobufError(err error) bool {
	_, ok := err.(*pberror)
	return ok
}

// ErrorOption is an option for the NewError constructor.
type ErrorOption struct{ apply func(*pberror) }

// WithDetails sets the details of the error.
func WithDetails(details ...proto.Message) ErrorOption {
	return ErrorOption{func(err *pberror) {
		if len(details) != 0 {
			err.details = details
		}
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
		if marshalErr != nil {
			return marshalErr
		}
		yarpcDet := make([]byte, len(detailsBytes))
		copy(yarpcDet, detailsBytes)
		cleanup()
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

	return NewError(yarpcErr.Code(), yarpcErr.Message(), WithDetails(protobufDetails...))
}
