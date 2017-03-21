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

package protobuf

import (
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"
)

// protoMarshal calls proto.Marshal but wraps checking for proto.ErrNil.
//
// there is a bug in golang/protobuf where message != nil && err == proto.ErrNil
func protoMarshal(message proto.Message) ([]byte, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		if err == proto.ErrNil {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

// toProcedureName gets the procedure name we should use for a protobuf method with the given service and name.
func toProcedureName(service string, method string) string {
	return fmt.Sprintf("%s::%s", service, method)
}

// fromProcedureName splits the given procedure name into the protobuf service name and method.
func fromProcedureName(procedureName string) (service, method string) {
	parts := strings.SplitN(procedureName, "::", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
