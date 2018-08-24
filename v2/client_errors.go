package yarpctransport

import (
	"fmt"
	"strings"

	"go.uber.org/yarpc/v2/yarpcerrors"
)

// RequestBodyEncodeError builds a YARPC error with code
// yarpcerrors.CodeInvalidArgument that represents a failure to encode
// the request body.
func RequestBodyEncodeError(req *Request, err error) error {
	return newClientEncodingError(req, false /*isResponse*/, false /*isHeader*/, err)
}

// ResponseBodyDecodeError builds a YARPC error with code
// yarpcerrors.CodeInvalidArgument that represents a failure to decode
// the response body.
func ResponseBodyDecodeError(req *Request, err error) error {
	return newClientEncodingError(req, true /*isResponse*/, false /*isHeader*/, err)
}

// RequestHeadersEncodeError builds a YARPC error with code
// yarpcerrors.CodeInvalidArgument that represents a failure to
// encode the request headers.
func RequestHeadersEncodeError(req *Request, err error) error {
	return newClientEncodingError(req, false /*isResponse*/, true /*isHeader*/, err)
}

// ResponseHeadersDecodeError builds a YARPC error with code
// yarpcerrors.CodeInvalidArgument that represents a failure to
// decode the response headers.
func ResponseHeadersDecodeError(req *Request, err error) error {
	return newClientEncodingError(req, true /*isResponse*/, true /*isHeader*/, err)
}

func newClientEncodingError(req *Request, isResponse bool, isHeader bool, err error) error {
	parts := []string{"failed to"}
	if isResponse {
		parts = append(parts, fmt.Sprintf("decode %q response", string(req.Encoding)))
	} else {
		parts = append(parts, fmt.Sprintf("encode %q request", string(req.Encoding)))
	}
	if isHeader {
		parts = append(parts, "headers")
	} else {
		parts = append(parts, "body")
	}
	parts = append(parts,
		fmt.Sprintf("for procedure %q of service %q: %v",
			req.Procedure, req.Service, err))
	return yarpcerrors.Newf(yarpcerrors.CodeInvalidArgument, strings.Join(parts, " "))
}
