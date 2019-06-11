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
)
