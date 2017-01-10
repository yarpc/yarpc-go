// Copyright (c) 2017 Uber Technologies, Inc.
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

package main

import (
	"strings"

	"go.uber.org/thriftrw/plugin"
	"go.uber.org/thriftrw/plugin/api"
)

// templateData contains all the data needed for the different code gen
// templates used by this plugin.
type templateData struct {
	Module       *api.Module
	Service      *api.Service
	Parent       *api.Service
	ParentModule *api.Module

	ContextImportPath   string
	UnaryWrapperImport  string
	UnaryWrapperFunc    string
	OnewayWrapperImport string
	OnewayWrapperFunc   string
}

// genFunc is a function that generates some part of the code needed by the
// plugin.
type genFunc func(*templateData, map[string][]byte) error

// Default options for the template
var templateOptions = []plugin.TemplateOption{
	plugin.TemplateFunc("lower", strings.ToLower),
}
