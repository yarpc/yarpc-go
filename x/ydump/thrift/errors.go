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

package thrift

import (
	"fmt"
	"strings"

	"go.uber.org/thriftrw/wire"
)

type fieldGroupError struct {
	available       []string
	missingRequired []string
	notFound        []string
}

func (e *fieldGroupError) addNotFound(arg string) {
	e.notFound = append(e.notFound, arg)
}

func (e *fieldGroupError) addMissingRequired(arg string) {
	e.missingRequired = append(e.missingRequired, arg)
}

func (e fieldGroupError) Error() string {
	messages := []string{"failed to parse fields"}
	if len(e.missingRequired) > 0 {
		messages = append(messages,
			messageList("the following fields are required but not specified", e.missingRequired))
	}

	if len(e.notFound) > 0 {
		messages = append(messages,
			messageList("the following fields were specified but not found", e.notFound))
		messages = append(messages,
			messageList("the available fields are", e.available))
	}

	return strings.Join(messages, "\n")
}

func (e fieldGroupError) asError() error {
	if len(e.missingRequired) == 0 && len(e.notFound) == 0 {
		return nil
	}

	return e
}

type specTypeMismatch struct {
	specified wire.Type
	got       wire.Type
}

func (e specTypeMismatch) Error() string {
	return fmt.Sprintf("type specified in Thrift field as %v, got %v", e.specified, e.got)
}

type specValueMismatch struct {
	specName   string
	underlying error
}

func (e specValueMismatch) Error() string {
	return fmt.Sprintf("field %q failed: %v", e.specName, e.underlying)
}

type specListItemMismatch struct {
	index      int
	underlying error
}

func (e specListItemMismatch) Error() string {
	return fmt.Sprintf("item %v failed: %v", e.index, e.underlying)
}

type specMapItemMismatch struct {
	specType   string
	underlying error
}

func (e specMapItemMismatch) Error() string {
	return fmt.Sprintf("%v failed: %v", e.specType, e.underlying)
}

type specStructFieldMismatch struct {
	fieldName  string
	underlying error
}

func (e specStructFieldMismatch) Error() string {
	return fmt.Sprintf("%q failed: %v", e.fieldName, e.underlying)
}

// messageList formats a message and a list for an error message.
func messageList(message string, list []string) string {
	if len(list) == 0 {
		return ""
	}

	newList := make([]string, len(list)+1)
	newList[0] = message
	copy(newList[1:], list)
	return strings.Join(newList, "\n\t")
}
