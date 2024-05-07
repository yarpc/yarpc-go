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

type responseWriterConstructor func(inboundCallResponse, tchannel.Format, headerCase) responseWriter

type responseWriterImpl struct {
	failedWith       error
	format           tchannel.Format
	headers          transport.Headers
	buffer           *bufferpool.Buffer
	response         inboundCallResponse
	applicationError bool
	headerCase       headerCase
}

func newHandlerWriter(response inboundCallResponse, format tchannel.Format, headerCase headerCase) responseWriter {
	return &responseWriterImpl{
		response:   response,
		format:     format,
		headerCase: headerCase,
	}
}

func (w *responseWriterImpl) AddHeaders(h transport.Headers) {
	for k, v := range h.OriginalItems() {
		if isReservedHeaderKey(k) {
			w.failedWith = appendError(w.failedWith, fmt.Errorf("cannot use reserved header key: %s", k))
			return
		}
		w.addHeader(k, v)
	}
}

func (w *responseWriterImpl) AddSystemHeader(key, value string) {
	w.addHeader(key, value)
}

func (w *responseWriterImpl) addHeader(key, value string) {
	w.headers = w.headers.With(key, value)
}

func (w *responseWriterImpl) SetApplicationError() {
	w.applicationError = true
}

func (w *responseWriterImpl) SetApplicationErrorMeta(applicationErrorMeta *transport.ApplicationErrorMeta) {
	if applicationErrorMeta == nil {
		return
	}
	if applicationErrorMeta.Code != nil {
		w.AddSystemHeader(ApplicationErrorCodeHeaderKey, strconv.Itoa(int(*applicationErrorMeta.Code)))
	}
	if applicationErrorMeta.Name != "" {
		w.AddSystemHeader(ApplicationErrorNameHeaderKey, applicationErrorMeta.Name)
	}
	if applicationErrorMeta.Details != "" {
		w.AddSystemHeader(ApplicationErrorDetailsHeaderKey, truncateAppErrDetails(applicationErrorMeta.Details))
	}
}

func truncateAppErrDetails(val string) string {
	if len(val) <= _maxAppErrDetailsHeaderLen {
		return val
	}
	stripIndex := _maxAppErrDetailsHeaderLen - len(_truncatedHeaderMessage)
	return val[:stripIndex] + _truncatedHeaderMessage
}

func (w *responseWriterImpl) IsApplicationError() bool {
	return w.applicationError
}

func (w *responseWriterImpl) Write(s []byte) (int, error) {
	if w.failedWith != nil {
		return 0, w.failedWith
	}

	if w.buffer == nil {
		w.buffer = bufferpool.Get()
	}

	n, err := w.buffer.Write(s)
	if err != nil {
		w.failedWith = appendError(w.failedWith, err)
	}
	return n, err
}

func (w *responseWriterImpl) Close() error {
	retErr := w.failedWith
	if w.IsApplicationError() {
		if err := w.response.SetApplicationError(); err != nil {
			retErr = appendError(retErr, fmt.Errorf("SetApplicationError() failed: %v", err))
		}
	}

	headers := getHeaderMap(w.headers, w.headerCase)
	retErr = appendError(retErr, writeHeaders(w.format, headers, nil, w.response.Arg2Writer))

	// Arg3Writer must be opened and closed regardless of if there is data
	// However, if there is a system error, we do not want to do this
	bodyWriter, err := w.response.Arg3Writer()
	if err != nil {
		return appendError(retErr, err)
	}
	defer func() { retErr = appendError(retErr, bodyWriter.Close()) }()
	if w.buffer != nil {
		if _, err := w.buffer.WriteTo(bodyWriter); err != nil {
			return appendError(retErr, err)
		}
	}

	return retErr
}

func (w *responseWriterImpl) ReleaseBuffer() {
	if w.buffer != nil {
		bufferpool.Put(w.buffer)
		w.buffer = nil
	}
}
