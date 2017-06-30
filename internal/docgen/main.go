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

// Package main is an internal program that generates the yarpc-go documentation
// in the docs directory from the templates in docs/templates.
//
// The program provides a struct that has all information all docs need. The caller
// should pass a given template to stdin and the resulting documentation will be on stdout.
//
// Example: go run internal/docgen/main.go < docs/templates/errors.md > docs/errors.md
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"text/template"

	"go.uber.org/yarpc/api/yarpcerrors"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/yarpc/transport/x/grpc"

	exttchannel "github.com/uber/tchannel-go"
	"google.golang.org/grpc/codes"
)

func main() {
	flag.Parse()
	if err := do(); err != nil {
		log.Fatal(err)
	}
}

func do() error {
	tmplData, err := getTmplData()
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	tmpl, err := template.New("tmpl").Parse(string(data))
	if err != nil {
		return err
	}
	if err := tmpl.Execute(os.Stdout, tmplData); err != nil {
		return err
	}
	return nil
}

func getTmplData() (interface{}, error) {
	return struct {
		CodeToGRPCCode       map[yarpcerrors.Code]codes.Code
		CodeToHTTPStatusCode map[yarpcerrors.Code]int
		CodeToTChannelCode   map[yarpcerrors.Code]exttchannel.SystemErrCode
	}{
		CodeToGRPCCode:       grpc.CodeToGRPCCode,
		CodeToHTTPStatusCode: http.CodeToStatusCode,
		CodeToTChannelCode:   tchannel.CodeToTChannelCode,
	}, nil
}
