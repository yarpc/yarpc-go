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

package protoplugin

import (
	"errors"
	"fmt"

	"github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"go.uber.org/multierr"
)

type multiRunner struct {
	runners []Runner
}

func newMultiRunner(runners ...Runner) *multiRunner {
	return &multiRunner{runners: runners}
}

func (m *multiRunner) Run(request *plugin_go.CodeGeneratorRequest) *plugin_go.CodeGeneratorResponse {
	nameToFile := make(map[string]*plugin_go.CodeGeneratorResponse_File)
	var responseErr error
	for _, runner := range m.runners {
		response := runner.Run(request)
		if responseErrString := response.GetError(); responseErrString != "" {
			responseErr = multierr.Append(responseErr, errors.New(responseErrString))
			continue
		}
		for _, file := range response.GetFile() {
			name := file.GetName()
			if name == "" {
				return newResponseError(errors.New("no name on CodeGeneratorResponse_File"))
			}
			if _, ok := nameToFile[name]; ok {
				return newResponseError(fmt.Errorf("duplicate name for CodeGeneratorResponse_File: %s", name))
			}
			nameToFile[name] = file
		}
	}
	if responseErr != nil {
		return newResponseError(responseErr)
	}
	files := make([]*plugin_go.CodeGeneratorResponse_File, 0, len(nameToFile))
	for _, file := range nameToFile {
		files = append(files, file)
	}
	return newResponseFiles(files)
}
