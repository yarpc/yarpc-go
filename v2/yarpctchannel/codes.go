// Copyright (c) 2018 Uber Technologies, Inc.
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

package yarpctchannel

import (
	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/v2/yarpcerror"
)

var (
	// _codeToTChannelCode maps Codes to their corresponding TChannel SystemErrCode.
	//
	// This only covers system-level errors, so if the code is not in the map,
	// it should not result in TChannel returning a system-level error.
	_codeToTChannelCode = map[yarpcerror.Code]tchannel.SystemErrCode{
		yarpcerror.CodeCancelled:         tchannel.ErrCodeCancelled,
		yarpcerror.CodeUnknown:           tchannel.ErrCodeUnexpected,
		yarpcerror.CodeInvalidArgument:   tchannel.ErrCodeBadRequest,
		yarpcerror.CodeDeadlineExceeded:  tchannel.ErrCodeTimeout,
		yarpcerror.CodeUnimplemented:     tchannel.ErrCodeBadRequest,
		yarpcerror.CodeInternal:          tchannel.ErrCodeUnexpected,
		yarpcerror.CodeUnavailable:       tchannel.ErrCodeDeclined,
		yarpcerror.CodeDataLoss:          tchannel.ErrCodeProtocol,
		yarpcerror.CodeResourceExhausted: tchannel.ErrCodeBusy,
	}

	// _tchannelCodeToCode maps TChannel SystemErrCodes to their corresponding Code.
	_tchannelCodeToCode = map[tchannel.SystemErrCode]yarpcerror.Code{
		tchannel.ErrCodeTimeout:    yarpcerror.CodeDeadlineExceeded,
		tchannel.ErrCodeCancelled:  yarpcerror.CodeCancelled,
		tchannel.ErrCodeBusy:       yarpcerror.CodeUnavailable,
		tchannel.ErrCodeDeclined:   yarpcerror.CodeUnavailable,
		tchannel.ErrCodeUnexpected: yarpcerror.CodeInternal,
		tchannel.ErrCodeBadRequest: yarpcerror.CodeInvalidArgument,
		tchannel.ErrCodeNetwork:    yarpcerror.CodeUnavailable,
		tchannel.ErrCodeProtocol:   yarpcerror.CodeInternal,
	}
)
