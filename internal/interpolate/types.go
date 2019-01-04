// Copyright (c) 2019 Uber Technologies, Inc.
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

package interpolate

import (
	"bytes"
	"fmt"
	"io"
)

// We represent the user-defined string as a series of terms. Each term is
// either a literal or a variable. Literals are used as-is and variables are
// resolved using a VariableResolver.
type (
	term interface {
		term()
	}

	literal string

	variable struct {
		Name       string
		Default    string
		HasDefault bool
	}
)

func (literal) term()  {}
func (variable) term() {}

// VariableResolver resolves the value of a variable specified in the string.
//
// The boolean value indicates whether this variable had a value defined. If a
// variable does not have a value and no default is specified, rendering will
// fail.
type VariableResolver func(name string) (value string, ok bool)

// String is a string that supports interpolation given some source of
// variable values.
//
// A String can be obtained by calling Parse on a string.
type String []term

// Render renders and returns the string. The provided VariableResolver will
// be used to determine values for the different variables mentioned in the
// string.
func (s String) Render(resolve VariableResolver) (string, error) {
	var buff bytes.Buffer
	if err := s.RenderTo(&buff, resolve); err != nil {
		return "", err
	}
	return buff.String(), nil
}

// RenderTo renders the string into the given writer. The provided
// VariableResolver will be used to determine values for the different
// variables mentioned in the string.
func (s String) RenderTo(w io.Writer, resolve VariableResolver) error {
	for _, term := range s {
		var value string
		switch t := term.(type) {
		case literal:
			value = string(t)
		case variable:
			if val, ok := resolve(t.Name); ok {
				value = val
			} else if t.HasDefault {
				value = t.Default
			} else {
				return errUnknownVariable{Name: t.Name}
			}
		}
		if _, err := io.WriteString(w, value); err != nil {
			return err
		}
	}
	return nil
}

type errUnknownVariable struct{ Name string }

func (e errUnknownVariable) Error() string {
	return fmt.Sprintf("variable %q does not have a value or a default", e.Name)
}
