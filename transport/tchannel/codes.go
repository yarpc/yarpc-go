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

package tchannel

import (
	"go.uber.org/yarpc/api/yarpcerrors"

	"github.com/uber/tchannel-go"
)

var (
	// CodeToTChannelCode maps Codes to their corresponding TChannel SystemErrCode.
	//
	// This only covers system-level errors, so if the code is not in the map,
	// it should not result in TChannel returning a system-level error.
	CodeToTChannelCode = map[yarpcerrors.Code]tchannel.SystemErrCode{
		// this is a 400-level code, what to do?
		yarpcerrors.CodeCancelled: tchannel.ErrCodeCancelled,
		yarpcerrors.CodeUnknown:   tchannel.ErrCodeUnexpected,
		// this is a 400-level code, what to do?
		yarpcerrors.CodeInvalidArgument:  tchannel.ErrCodeBadRequest,
		yarpcerrors.CodeDeadlineExceeded: tchannel.ErrCodeTimeout,
		yarpcerrors.CodeUnimplemented:    tchannel.ErrCodeBadRequest,
		yarpcerrors.CodeInternal:         tchannel.ErrCodeUnexpected,
		yarpcerrors.CodeUnavailable:      tchannel.ErrCodeNetwork,
		yarpcerrors.CodeDataLoss:         tchannel.ErrCodeUnexpected,
	}

	// TChannelCodeToCode maps TChannel SystemErrCodes to their corresponding Code.
	TChannelCodeToCode = map[tchannel.SystemErrCode]yarpcerrors.Code{
		tchannel.ErrCodeTimeout: yarpcerrors.CodeDeadlineExceeded,
		// this is a 400-level code, what to do?
		tchannel.ErrCodeCancelled:  yarpcerrors.CodeCancelled,
		tchannel.ErrCodeBusy:       yarpcerrors.CodeUnavailable,
		tchannel.ErrCodeDeclined:   yarpcerrors.CodeUnavailable,
		tchannel.ErrCodeUnexpected: yarpcerrors.CodeInternal,
		// this is a 400-level code, what to do?
		tchannel.ErrCodeBadRequest: yarpcerrors.CodeInvalidArgument,
		tchannel.ErrCodeNetwork:    yarpcerrors.CodeUnavailable,
		tchannel.ErrCodeProtocol:   yarpcerrors.CodeInternal,
	}
)
