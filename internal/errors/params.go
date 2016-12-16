package errors

import (
	"fmt"
	"strings"
)

// MissingParameters returns an error representing a failure to process a
// request because it was missing required parameters.
func MissingParameters(params []string) error {
	if len(params) == 0 {
		return nil
	}

	return missingParametersError{Parameters: params}
}

// missingParametersError is a failure to process a request because it was
// missing required parameters.
type missingParametersError struct {
	// Names of the missing parameters.
	//
	// Precondition: len(Parameters) > 0
	Parameters []string
}

func (e missingParametersError) AsHandlerError() HandlerError {
	return HandlerBadRequestError(e)
}

func (e missingParametersError) Error() string {
	s := "missing "
	ps := e.Parameters
	if len(ps) == 1 {
		s += ps[0]
		return s
	}

	if len(ps) == 2 {
		s += fmt.Sprintf("%s and %s", ps[0], ps[1])
		return s
	}

	s += strings.Join(ps[:len(ps)-1], ", ")
	s += fmt.Sprintf(", and %s", ps[len(ps)-1])
	return s
}
