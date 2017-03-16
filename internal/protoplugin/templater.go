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

package protoplugin

import (
	"bytes"
	"text/template"
)

//type Templater interface {
//Apply(templateInfo *TemplateInfo) (string, error)
//}

//func NewTemplater(tmpl *template.Template) Templater {
//return newTemplater(tmpl)
//}

type templater struct {
	tmpl *template.Template
}

func newTemplater(tmpl *template.Template) *templater {
	return &templater{tmpl}
}

func (t *templater) Apply(templateInfo *TemplateInfo) (string, error) {
	buffer := bytes.NewBuffer(nil)
	if err := t.tmpl.Execute(buffer, templateInfo); err != nil {
		return "", err
	}
	return buffer.String(), nil
}
