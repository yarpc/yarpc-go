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

package protoplugin

import (
	"errors"
	"testing"

	"github.com/gogo/protobuf/proto"
	plugin_go "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/multierr"
)

func TestMultiRunnerRun(t *testing.T) {
	testMultiRunnerRun(
		t,
		[]Runner{
			newSuccessTestRunner("foo"),
			newSuccessTestRunner("bar", "baz"),
			newSuccessTestRunner("bat"),
		},
		[]string{
			"bar",
			"bat",
			"baz",
			"foo",
		},
		nil,
	)
	testMultiRunnerRun(
		t,
		[]Runner{
			newSuccessTestRunner("foo"),
			newSuccessTestRunner("bar", "baz"),
			newSuccessTestRunner(""),
		},
		nil,
		errNoFileName,
	)
	testMultiRunnerRun(
		t,
		[]Runner{
			newSuccessTestRunner("foo"),
			newSuccessTestRunner("bar", "baz"),
			newSuccessTestRunner("foo"),
		},
		nil,
		newErrorDuplicateFileName("foo"),
	)
	testMultiRunnerRun(
		t,
		[]Runner{
			newSuccessTestRunner("foo"),
			newErrorTestRunner(errors.New("value")),
			newSuccessTestRunner("bar"),
		},
		nil,
		errors.New("value"),
	)
	testMultiRunnerRun(
		t,
		[]Runner{
			newSuccessTestRunner("foo"),
			newErrorTestRunner(errors.New("value")),
			newErrorTestRunner(errors.New("value2")),
		},
		nil,
		multierr.Append(
			errors.New("value"),
			errors.New("value2"),
		),
	)
}

func testMultiRunnerRun(
	t *testing.T,
	runners []Runner,
	expectedFileNames []string,
	expectedError error,
) {
	response := NewMultiRunner(runners...).Run(nil)
	if expectedError != nil {
		assert.Equal(t, newResponseError(expectedError), response)
	} else {
		fileNames := make([]string, 0, len(expectedFileNames))
		for _, file := range response.GetFile() {
			fileNames = append(fileNames, file.GetName())
		}
		assert.Equal(t, expectedFileNames, fileNames)
	}
}

type testRunner struct {
	fileNames []string
	err       error
}

func newSuccessTestRunner(fileNames ...string) *testRunner {
	return &testRunner{
		fileNames: fileNames,
	}
}

func newErrorTestRunner(err error) *testRunner {
	return &testRunner{
		err: err,
	}
}

func (r *testRunner) Run(*plugin_go.CodeGeneratorRequest) *plugin_go.CodeGeneratorResponse {
	if r.err != nil {
		return newResponseError(r.err)
	}
	files := make([]*plugin_go.CodeGeneratorResponse_File, 0, len(r.fileNames))
	for _, fileName := range r.fileNames {
		files = append(files, &plugin_go.CodeGeneratorResponse_File{
			Name: proto.String(fileName),
		})
	}
	return newResponseFiles(files)
}
