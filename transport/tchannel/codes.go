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
	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
)

var (
	// CodeToTChannelCode maps Codes to their corresponding TChannel SystemErrCode.
	//
	// This only covers system-level errors, so if the code is not in the map,
	// it should not result in TChannel returning a system-level error.
	CodeToTChannelCode = map[transport.Code]tchannel.SystemErrCode{
		yarpc.CodeCancelled:         tchannel.ErrCodeCancelled,
		yarpc.CodeUnknown:           tchannel.ErrCodeUnexpected,
		yarpc.CodeInvalidArgument:   tchannel.ErrCodeBadRequest,
		yarpc.CodeDeadlineExceeded:  tchannel.ErrCodeTimeout,
		yarpc.CodeUnimplemented:     tchannel.ErrCodeBadRequest,
		yarpc.CodeInternal:          tchannel.ErrCodeUnexpected,
		yarpc.CodeUnavailable:       tchannel.ErrCodeDeclined,
		yarpc.CodeDataLoss:          tchannel.ErrCodeProtocol,
		yarpc.CodeResourceExhausted: tchannel.ErrCodeBusy,
	}

	// TChannelCodeToCode maps TChannel SystemErrCodes to their corresponding Code.
	TChannelCodeToCode = map[tchannel.SystemErrCode]transport.Code{
		tchannel.ErrCodeTimeout:    yarpc.CodeDeadlineExceeded,
		tchannel.ErrCodeCancelled:  yarpc.CodeCancelled,
		tchannel.ErrCodeBusy:       yarpc.CodeUnavailable,
		tchannel.ErrCodeDeclined:   yarpc.CodeUnavailable,
		tchannel.ErrCodeUnexpected: yarpc.CodeInternal,
		tchannel.ErrCodeBadRequest: yarpc.CodeInvalidArgument,
		tchannel.ErrCodeNetwork:    yarpc.CodeUnavailable,
		tchannel.ErrCodeProtocol:   yarpc.CodeInternal,
	}
)
