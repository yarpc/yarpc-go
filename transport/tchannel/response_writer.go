// Copyright (c) 2024 Uber Technologies, Inc.
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
	"fmt"
	"strconv"

	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/bufferpool"
)

type responseWriterImpl struct {
	failedWith       error
	format           tchannel.Format
	headers          transport.Headers
	buffer           *bufferpool.Buffer
	response         inboundCallResponse
	applicationError bool
	headerCase       headerCase
	reservedHeader   bool
}

func newResponseWriter(response inboundCallResponse, format tchannel.Format, headerCase headerCase) responseWriter {
	return &responseWriterImpl{
		response:   response,
		format:     format,
		headerCase: headerCase,
	}
}

func (hw *responseWriterImpl) AddHeaders(h transport.Headers) {
	for k, v := range h.OriginalItems() {
		if !isReservedHeaderPrefix(k) {
			hw.addHeader(k, v)
			continue
		}

		hw.reservedHeader = true
		if enforceHeaderRules {
			hw.failedWith = appendError(hw.failedWith, fmt.Errorf("header with rpc prefix is not allowed in response application headers (%s was passed)", k))
			return
		} else if isReservedHeaderKey(k) {
			hw.failedWith = appendError(hw.failedWith, fmt.Errorf("cannot use reserved header key: %s", k))
			return
		} else {
			hw.addHeader(k, v)
		}
	}
}

func (hw *responseWriterImpl) AddSystemHeader(key, value string) {
	hw.addHeader(key, value)
}

func (hw *responseWriterImpl) addHeader(key, value string) {
	hw.headers = hw.headers.With(key, value)
}

func (hw *responseWriterImpl) SetApplicationError() {
	hw.applicationError = true
}

func (hw *responseWriterImpl) SetApplicationErrorMeta(applicationErrorMeta *transport.ApplicationErrorMeta) {
	if applicationErrorMeta == nil {
		return
	}
	if applicationErrorMeta.Code != nil {
		hw.AddSystemHeader(ApplicationErrorCodeHeaderKey, strconv.Itoa(int(*applicationErrorMeta.Code)))
	}
	if applicationErrorMeta.Name != "" {
		hw.AddSystemHeader(ApplicationErrorNameHeaderKey, applicationErrorMeta.Name)
	}
	if applicationErrorMeta.Details != "" {
		hw.AddSystemHeader(ApplicationErrorDetailsHeaderKey, truncateAppErrDetails(applicationErrorMeta.Details))
	}
}

func truncateAppErrDetails(val string) string {
	if len(val) <= _maxAppErrDetailsHeaderLen {
		return val
	}
	stripIndex := _maxAppErrDetailsHeaderLen - len(_truncatedHeaderMessage)
	return val[:stripIndex] + _truncatedHeaderMessage
}

func (hw *responseWriterImpl) IsApplicationError() bool {
	return hw.applicationError
}

func (hw *responseWriterImpl) Write(s []byte) (int, error) {
	if hw.failedWith != nil {
		return 0, hw.failedWith
	}

	if hw.buffer == nil {
		hw.buffer = bufferpool.Get()
	}

	n, err := hw.buffer.Write(s)
	if err != nil {
		hw.failedWith = appendError(hw.failedWith, err)
	}
	return n, err
}

func (hw *responseWriterImpl) Close() error {
	retErr := hw.failedWith
	if hw.IsApplicationError() {
		if err := hw.response.SetApplicationError(); err != nil {
			retErr = appendError(retErr, fmt.Errorf("SetApplicationError() failed: %v", err))
		}
	}

	headers := getHeaderMap(hw.headers, hw.headerCase)
	retErr = appendError(retErr, writeHeaders(hw.format, headers, nil, hw.response.Arg2Writer))

	// Arg3Writer must be opened and closed regardless of if there is data
	// However, if there is a system error, we do not want to do this
	bodyWriter, err := hw.response.Arg3Writer()
	if err != nil {
		return appendError(retErr, err)
	}
	defer func() { retErr = appendError(retErr, bodyWriter.Close()) }()
	if hw.buffer != nil {
		if _, err := hw.buffer.WriteTo(bodyWriter); err != nil {
			return appendError(retErr, err)
		}
	}

	return retErr
}

func (hw *responseWriterImpl) ReleaseBuffer() {
	if hw.buffer != nil {
		bufferpool.Put(hw.buffer)
		hw.buffer = nil
	}
}

func (hw *responseWriterImpl) IsReservedHeaderUsed() bool {
	return hw.reservedHeader
}
