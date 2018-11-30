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

package yarpchttp

import "time"

const transportName = "http"

var (
	defaultConnTimeout     = 500 * time.Millisecond
	defaultInnocenceWindow = 5 * time.Second
)

// HTTP headers used in requests and responses to send YARPC metadata.
const (
	// PeerHeader is the header key for carrying the return address for a
	// request or response.
	PeerHeader = "Rpc-Peer"

	// Name of the service sending the request. This corresponds to the
	// Request.Caller attribute.
	CallerHeader = "Rpc-Caller"

	// Name of the encoding used for the request body. This corresponds to the
	// Request.Encoding attribute.
	EncodingHeader = "Rpc-Encoding"

	// Amount of time (in milliseconds) within which the request is expected
	// to finish.
	TTLMSHeader = "Context-TTL-MS"

	// Name of the procedure being called. This corresponds to the
	// Request.Procedure attribute.
	ProcedureHeader = "Rpc-Procedure"

	// Name of the service to which the request is being sent. This
	// corresponds to the Request.Service attribute. This header is also used
	// in responses to ensure requests are processed by the correct service.
	ServiceHeader = "Rpc-Service"

	// Shard key used by the destined service to shard the request. This
	// corresponds to the Request.ShardKey attribute.
	ShardKeyHeader = "Rpc-Shard-Key"

	// The traffic group responsible for handling the request. This
	// corresponds to the Request.RoutingKey attribute.
	RoutingKeyHeader = "Rpc-Routing-Key"

	// A service that can proxy the destined service. This corresponds to the
	// Request.RoutingDelegate attribute.
	RoutingDelegateHeader = "Rpc-Routing-Delegate"

	// Whether the response body contains an application error.
	ApplicationStatusHeader = "Rpc-Status"

	// ErrorCodeHeader contains the string representation of the error code.
	ErrorCodeHeader = "Rpc-Error-Code"

	// ErrorNameHeader contains the name of a user-defined error.
	ErrorNameHeader = "Rpc-Error-Name"

	// ErrorMessageHeader contains the message of an error, if the
	// BothResponseError feature is enabled.
	ErrorMessageHeader = "Rpc-Error-Message"

	// AcceptsBothResponseErrorHeader says that the BothResponseError
	// feature is supported on the client. If the value is "true",
	// this indicates true.
	AcceptsBothResponseErrorHeader = "Rpc-Accepts-Both-Response-Error"

	// BothResponseErrorHeader says that the BothResponseError
	// feature is supported on the server. If any non-empty value is set,
	// this indicates true.
	BothResponseErrorHeader = "Rpc-Both-Response-Error"

	// ContentTypeHeader is the key of the HTTP Content-Type header.
	ContentTypeHeader = "Content-Type"

	// TextPlainHeader is the text/plain Content-Type header value.
	TextPlainHeader = "text/plain; charset=utf8"
)

// Valid values for the Rpc-Status header.
const (
	// The request was successful.
	ApplicationSuccessStatus = "success"

	// An error occurred. The response body contains an application header.
	ApplicationErrorStatus = "error"

	// AcceptTrue is the true value used for accept headers.
	AcceptTrue = "true"
)

// ApplicationHeaderPrefix is the prefix added to application header keys to
// send them in requests or responses.
const ApplicationHeaderPrefix = "Rpc-Header-"