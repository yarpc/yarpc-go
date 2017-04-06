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

package grpc

import (
	"fmt"
	"net/url"
	"strings"

	"go.uber.org/yarpc/internal/procedure"
)

const defaultMethodName = "__default__"

func procedureNameToServiceNameMethodName(procedureName string) (string, string, error) {
	// TODO: this is hacky
	serviceName, methodName := procedure.FromName(strings.Replace(procedureName, "/", "___", -1))
	if serviceName == "" {
		return "", "", fmt.Errorf("invalid procedure name: %s", procedureName)
	}
	if methodName == "" {
		methodName = defaultMethodName
	}
	// TODO: do we really need to do url.QueryEscape?
	// Are there consequences if there is a diff from the string and the url.QueryEscape string?
	return url.QueryEscape(serviceName), url.QueryEscape(methodName), nil
}

func procedureNameToFullMethod(procedureName string) (string, error) {
	serviceName, methodName, err := procedureNameToServiceNameMethodName(procedureName)
	if err != nil {
		return "", err
	}
	return toFullMethod(serviceName, methodName), nil
}

func toFullMethod(serviceName string, methodName string) string {
	if methodName == "" {
		methodName = defaultMethodName
	}
	return fmt.Sprintf("/%s/%s", serviceName, methodName)
}

func procedureToName(serviceName string, methodName string) string {
	if methodName == defaultMethodName {
		// TODO: this is hacky
		return strings.Replace(serviceName, "___", "/", -1)
	}
	// TODO: this is hacky
	return strings.Replace(procedure.ToName(serviceName, methodName), "___", "/", -1)
}
