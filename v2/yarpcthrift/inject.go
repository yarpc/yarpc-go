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

package yarpcthrift

import (
	"reflect"
	"strings"

	"go.uber.org/yarpc/api/transport"
)

// ClientBuilderOptions returns ClientOptions that InjectClients should use
// for a specific Thrift client given information about the field into which
// the client is being injected. This API will usually not be used directly by
// users but by the generated code.
func ClientBuilderOptions(_ transport.ClientConfig, f reflect.StructField) []ClientOption {
	// Note that we don't use ClientConfig right now but since this code is
	// called by generated code, we still accept it so that we can add logic
	// based on it in the future without breaking the API (and thus, all
	// generated code).

	optionList := strings.Split(f.Tag.Get("thrift"), ",")
	var opts []ClientOption
	for _, opt := range optionList {
		switch strings.ToLower(opt) {
		case "multiplexed":
			opts = append(opts, Multiplexed)
		case "enveloped":
			opts = append(opts, Enveloped)
		default:
			// Ignore unknown options
		}
	}
	return opts
}
