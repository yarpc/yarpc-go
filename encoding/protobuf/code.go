package protobuf

import (
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc/codes"
)

var (
	// _codeToGRPCCode maps all Codes to their corresponding gRPC Code.
	_codeToGRPCCode = map[yarpcerrors.Code]codes.Code{
		yarpcerrors.CodeOK:                 codes.OK,
		yarpcerrors.CodeCancelled:          codes.Canceled,
		yarpcerrors.CodeUnknown:            codes.Unknown,
		yarpcerrors.CodeInvalidArgument:    codes.InvalidArgument,
		yarpcerrors.CodeDeadlineExceeded:   codes.DeadlineExceeded,
		yarpcerrors.CodeNotFound:           codes.NotFound,
		yarpcerrors.CodeAlreadyExists:      codes.AlreadyExists,
		yarpcerrors.CodePermissionDenied:   codes.PermissionDenied,
		yarpcerrors.CodeResourceExhausted:  codes.ResourceExhausted,
		yarpcerrors.CodeFailedPrecondition: codes.FailedPrecondition,
		yarpcerrors.CodeAborted:            codes.Aborted,
		yarpcerrors.CodeOutOfRange:         codes.OutOfRange,
		yarpcerrors.CodeUnimplemented:      codes.Unimplemented,
		yarpcerrors.CodeInternal:           codes.Internal,
		yarpcerrors.CodeUnavailable:        codes.Unavailable,
		yarpcerrors.CodeDataLoss:           codes.DataLoss,
		yarpcerrors.CodeUnauthenticated:    codes.Unauthenticated,
	}

	// _grpcCodeToCode maps all gRPC Codes to their corresponding Code.
	_grpcCodeToCode = map[codes.Code]yarpcerrors.Code{
		codes.OK:                 yarpcerrors.CodeOK,
		codes.Canceled:           yarpcerrors.CodeCancelled,
		codes.Unknown:            yarpcerrors.CodeUnknown,
		codes.InvalidArgument:    yarpcerrors.CodeInvalidArgument,
		codes.DeadlineExceeded:   yarpcerrors.CodeDeadlineExceeded,
		codes.NotFound:           yarpcerrors.CodeNotFound,
		codes.AlreadyExists:      yarpcerrors.CodeAlreadyExists,
		codes.PermissionDenied:   yarpcerrors.CodePermissionDenied,
		codes.ResourceExhausted:  yarpcerrors.CodeResourceExhausted,
		codes.FailedPrecondition: yarpcerrors.CodeFailedPrecondition,
		codes.Aborted:            yarpcerrors.CodeAborted,
		codes.OutOfRange:         yarpcerrors.CodeOutOfRange,
		codes.Unimplemented:      yarpcerrors.CodeUnimplemented,
		codes.Internal:           yarpcerrors.CodeInternal,
		codes.Unavailable:        yarpcerrors.CodeUnavailable,
		codes.DataLoss:           yarpcerrors.CodeDataLoss,
		codes.Unauthenticated:    yarpcerrors.CodeUnauthenticated,
	}
)
