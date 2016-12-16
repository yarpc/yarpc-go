// Copyright (c) 2016 Uber Technologies, Inc.
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

package request

import (
	"fmt"
	"strings"

	"go.uber.org/yarpc/internal/errors"
)

// missingParametersError is a failure to process a request because it was
// missing required parameters.
type missingParametersError struct {
	// Names of the missing parameters.
	//
	// Precondition: len(Parameters) > 0
	Parameters []string
}

func (e missingParametersError) AsHandlerError() errors.HandlerError {
	return errors.HandlerBadRequestError(e)
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
