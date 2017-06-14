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

// TODO note does not cover all Codes, for now will return ErrCodeUnexpected
// TODO: need to review all of this

var (
	_codeToTChannelCode = map[yarpcerrors.Code]tchannel.SystemErrCode{
		//yarpcerrors.CodeOK: ,
		yarpcerrors.CodeCancelled:        tchannel.ErrCodeCancelled,
		yarpcerrors.CodeUnknown:          tchannel.ErrCodeUnexpected,
		yarpcerrors.CodeInvalidArgument:  tchannel.ErrCodeBadRequest,
		yarpcerrors.CodeDeadlineExceeded: tchannel.ErrCodeTimeout,
		//yarpcerrors.CodeNotFound: ,
		//yarpcerrors.CodeAlreadyExists: ,
		//yarpcerrors.CodePermissionDenied: ,
		//yarpcerrors.CodeResourceExhausted: ,
		//yarpcerrors.CodeFailedPrecondition: ,
		//yarpcerrors.CodeAborted: ,
		//yarpcerrors.CodeOutOfRange: ,
		//yarpcerrors.CodeUnimplemented: ,
		yarpcerrors.CodeInternal:    tchannel.ErrCodeUnexpected,
		yarpcerrors.CodeUnavailable: tchannel.ErrCodeNetwork,
		yarpcerrors.CodeDataLoss:    tchannel.ErrCodeUnexpected,
		//yarpcerrors.CodeUnauthenticated: ,
	}

	_tchannelCodeToCode = map[tchannel.SystemErrCode]yarpcerrors.Code{
		tchannel.ErrCodeTimeout:    yarpcerrors.CodeDeadlineExceeded,
		tchannel.ErrCodeCancelled:  yarpcerrors.CodeCancelled,
		tchannel.ErrCodeBusy:       yarpcerrors.CodeUnavailable,
		tchannel.ErrCodeDeclined:   yarpcerrors.CodeUnavailable,
		tchannel.ErrCodeUnexpected: yarpcerrors.CodeInternal,
		tchannel.ErrCodeBadRequest: yarpcerrors.CodeInvalidArgument,
		tchannel.ErrCodeNetwork:    yarpcerrors.CodeUnavailable,
		tchannel.ErrCodeProtocol:   yarpcerrors.CodeInternal,
	}
)

func codeToTChannelCode(code yarpcerrors.Code) tchannel.SystemErrCode {
	tchannelCode, ok := _codeToTChannelCode[code]
	if !ok {
		return tchannel.ErrCodeUnexpected
	}
	return tchannelCode
}

func tchannelCodeToCode(tchannelCode tchannel.SystemErrCode) yarpcerrors.Code {
	code, ok := _tchannelCodeToCode[tchannelCode]
	if !ok {
		// TODO: is this correct?
		return yarpcerrors.CodeUnknown
	}
	return code
}
