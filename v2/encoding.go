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

package yarpc

import (
	"context"
	"fmt"

	"go.uber.org/yarpc/v2/yarpcerror"
)

var _ UnaryTransportHandler = (*unaryTransportHandler)(nil)

// EncodingToTransportProcedures converts encoding-level procedures to transport-level procedures.
func EncodingToTransportProcedures(encodingProcedures []EncodingProcedure) []TransportProcedure {
	transportProcedures := make([]TransportProcedure, len(encodingProcedures))
	for i, p := range encodingProcedures {
		var spec TransportHandlerSpec
		switch p.HandlerSpec.Type() {
		case Unary:
			spec = NewUnaryTransportHandlerSpec(&unaryTransportHandler{p})
			// TODO: handle Streaming case
		default:
			panic(fmt.Sprintf("unsupported handler spec type: %v", p.HandlerSpec.Type()))
		}

		transportProcedures[i] = TransportProcedure{
			Name:        p.Name,
			Service:     p.Service,
			HandlerSpec: spec,
			Encoding:    p.Encoding,
			Signature:   p.Signature,
		}
	}

	return transportProcedures
}

// Allows encoding-level procedures to act as transport-level procedures.
type unaryTransportHandler struct {
	h EncodingProcedure
}

func (u *unaryTransportHandler) Handle(ctx context.Context, req *Request, reqBuf *Buffer) (*Response, *Buffer, error) {
	res := &Response{}
	ctx, call := NewInboundCall(ctx)
	if err := call.ReadFromRequest(req); err != nil {
		return nil, nil, err
	}

	codec := u.h.Codec()

	decodedBody, err := codec.Decode(reqBuf)
	if err != nil {
		return nil, nil, err
	}

	body, appErr := u.h.HandlerSpec.Unary().Handle(ctx, decodedBody)
	call.WriteToResponse(res)

	encodedBody, err := codec.Encode(body)
	if err != nil {
		return nil, nil, err
	}

	if appErr != nil {
		encodedError, err := codec.EncodeError(appErr)
		if err != nil {
			return nil, nil, err
		}

		errorInfo := yarpcerror.ExtractInfo(appErr)
		res.ApplicationErrorInfo = &errorInfo
		return res, encodedError, nil
	}

	return res, encodedBody, nil
}
