package errors

import "testing"

func TestCoverBrands(t *testing.T) {
	// sorted
	RemoteTimeoutError("").timeoutError()
	clientTimeoutError{}.clientError()
	clientTimeoutError{}.timeoutError()
	handlerBadRequestError{}.badRequestError()
	handlerBadRequestError{}.handlerError()
	handlerTimeoutError{}.handlerError()
	handlerTimeoutError{}.timeoutError()
	handlerUnexpectedError{}.handlerError()
	handlerUnexpectedError{}.unexpectedError()
	remoteBadRequestError("").badRequestError()
	remoteUnexpectedError("").unexpectedError()
	unrecognizedEncodingError{}.unrecognizedEncodingError()
	unrecognizedProcedureError{}.unrecognizedProcedureError()
}
