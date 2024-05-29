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
	"context"
	"encoding/binary"
	"io"
	"strings"

	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/tchannel/internal"
	"go.uber.org/yarpc/yarpcerrors"
)

const (
	/** Response headers **/

	// ErrorCodeHeaderKey is the response header key for the error code.
	ErrorCodeHeaderKey = "$rpc$-error-code"
	// ErrorNameHeaderKey is the response header key for the error name.
	ErrorNameHeaderKey = "$rpc$-error-name"
	// ErrorMessageHeaderKey is the response header key for the error message.
	ErrorMessageHeaderKey = "$rpc$-error-message"
	// ServiceHeaderKey is the response header key for the respond service
	ServiceHeaderKey = "$rpc$-service"
	// ApplicationErrorNameHeaderKey is the response header key for the application error name.
	ApplicationErrorNameHeaderKey = "$rpc$-application-error-name"
	// ApplicationErrorDetailsHeaderKey is the response header key for the
	// application error details string.
	ApplicationErrorDetailsHeaderKey = "$rpc$-application-error-details"
	// ApplicationErrorCodeHeaderKey is the response header key for the application error code.
	ApplicationErrorCodeHeaderKey = "$rpc$-application-error-code"

	/** Request headers **/

	// CallerProcedureHeader is the header key for the procedure of the caller making the request.
	CallerProcedureHeader = "rpc-caller-procedure"
)

var _reservedHeaderKeys = map[string]struct{}{
	ErrorCodeHeaderKey:               {},
	ErrorNameHeaderKey:               {},
	ErrorMessageHeaderKey:            {},
	ServiceHeaderKey:                 {},
	ApplicationErrorNameHeaderKey:    {},
	ApplicationErrorDetailsHeaderKey: {},
	ApplicationErrorCodeHeaderKey:    {},
	CallerProcedureHeader:            {},
}

func isReservedHeaderKey(key string) bool {
	_, ok := _reservedHeaderKeys[strings.ToLower(key)]
	return ok
}

// readRequestHeaders reads headers and baggage from an incoming request.
func readRequestHeaders(
	ctx context.Context,
	format tchannel.Format,
	getReader func() (tchannel.ArgReader, error),
) (context.Context, transport.Headers, error) {
	headers, err := readHeaders(format, getReader)
	if err != nil {
		return ctx, headers, err
	}
	return ctx, headers, nil
}

// readHeaders reads headers using the given function to get the arg reader.
//
// This may be used with the Arg2Reader functions on InboundCall and
// OutboundCallResponse.
//
// If the format is JSON, the headers are expected to be JSON encoded.
//
// This function always returns a non-nil Headers object in case of success.
func readHeaders(format tchannel.Format, getReader func() (tchannel.ArgReader, error)) (transport.Headers, error) {
	if format == tchannel.JSON {
		// JSON is special
		var headers map[string]string
		err := tchannel.NewArgReader(getReader()).ReadJSON(&headers)
		return transport.HeadersFromMap(headers), err
	}

	r, err := getReader()
	if err != nil {
		return transport.Headers{}, err
	}

	headers, err := decodeHeaders(r)
	if err != nil {
		return headers, err
	}

	return headers, r.Close()
}

var emptyMap = map[string]string{}

// writeHeaders writes the given headers using the given function to get the
// arg writer.
//
// This may be used with the Arg2Writer functions on OutboundCall and
// InboundCallResponse.
//
// If the format is JSON, the headers are JSON encoded.
func writeHeaders(format tchannel.Format, headers map[string]string, tracingBaggage map[string]string, getWriter func() (tchannel.ArgWriter, error)) error {
	merged := mergeHeaders(headers, tracingBaggage)
	if format == tchannel.JSON {
		// JSON is special
		if merged == nil {
			// We want to write "{}", not "null" for empty map.
			merged = emptyMap
		}
		return tchannel.NewArgWriter(getWriter()).WriteJSON(merged)
	}
	return tchannel.NewArgWriter(getWriter()).Write(encodeHeaders(merged))
}

// mergeHeaders will keep the last value if the same key appears multiple times
func mergeHeaders(m1, m2 map[string]string) map[string]string {
	if len(m1) == 0 {
		return m2
	}
	if len(m2) == 0 {
		return m1
	}
	// merge and return
	merged := make(map[string]string, len(m1)+len(m2))
	for k, v := range m1 {
		merged[k] = v
	}
	for k, v := range m2 {
		merged[k] = v
	}
	return merged
}

// decodeHeaders decodes headers using the format:
//
//	nh:2 (k~2 v~2){nh}
func decodeHeaders(r io.Reader) (transport.Headers, error) {
	reader := internal.NewReader(r)

	count := reader.ReadUint16()
	if count == 0 {
		return transport.Headers{}, reader.Err()
	}

	headers := transport.NewHeadersWithCapacity(int(count))
	for i := 0; i < int(count) && reader.Err() == nil; i++ {
		k := reader.ReadLen16String()
		v := reader.ReadLen16String()
		headers = headers.With(k, v)
	}

	return headers, reader.Err()
}

// headerCallerProcedureToRequest copies callerProcedure from headers to req.CallerProcedure
// and then deletes it from headers.
func headerCallerProcedureToRequest(req *transport.Request, headers *transport.Headers) *transport.Request {
	if callerProcedure, ok := headers.Get(CallerProcedureHeader); ok {
		req.CallerProcedure = callerProcedure
		headers.Del(CallerProcedureHeader)
		return req
	}
	return req
}

// requestCallerProcedureToHeader add callerProcedure header as an application header.
func requestCallerProcedureToHeader(req *transport.Request, reqHeaders map[string]string) map[string]string {
	if req.CallerProcedure == "" {
		return reqHeaders
	}

	if reqHeaders == nil {
		reqHeaders = make(map[string]string)
	}
	reqHeaders[CallerProcedureHeader] = req.CallerProcedure
	return reqHeaders
}

// encodeHeaders encodes headers using the format:
//
//	nh:2 (k~2 v~2){nh}
func encodeHeaders(hs map[string]string) []byte {
	if len(hs) == 0 {
		return []byte{0, 0} // nh = 2
	}

	size := 2 // nh:2
	for k, v := range hs {
		size += len(k) + 2 // k~2
		size += len(v) + 2 // v~2
	}

	out := make([]byte, size)

	i := 2
	binary.BigEndian.PutUint16(out, uint16(len(hs))) // nh:2
	for k, v := range hs {
		i += _putStr16(k, out[i:]) // k~2
		i += _putStr16(v, out[i:]) // v~2
	}

	return out
}

func headerMap(hs transport.Headers, headerCase headerCase) map[string]string {
	switch headerCase {
	case originalHeaderCase:
		return hs.OriginalItems()
	default:
		return hs.Items()
	}
}

func deleteReservedHeaders(headers transport.Headers) {
	for headerKey := range _reservedHeaderKeys {
		headers.Del(headerKey)
	}
}

// this check ensures that the service we're issuing a request to is the one
// responding
func validateServiceName(requestService, responseService string) error {
	// an empty service string means that we're talking to an older YARPC
	// TChannel client
	if responseService == "" || requestService == responseService {
		return nil
	}
	return yarpcerrors.InternalErrorf(
		"service name sent from the request does not match the service name "+
			"received in the response: sent %q, got: %q", requestService, responseService)
}

// _putStr16 writes the bytes `in` into `out` using the encoding `s~2`.
func _putStr16(in string, out []byte) int {
	binary.BigEndian.PutUint16(out, uint16(len(in)))
	return copy(out[2:], in) + 2
}
