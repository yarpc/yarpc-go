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

package introspection

import (
	"go.uber.org/yarpc/api/transport"
)

// Procedure represent a registered procedure on a dispatcher.
type Procedure struct {
	Name      string `json:"name"`
	Encoding  string `json:"encoding"`
	Signature string `json:"signature"`
	RPCType   string `json:"rpcType"`
}

// IntrospectProcedures is a convenience function that translate a slice of
// transport.Procedure to a slice of introspection.Procedure. This output is
// used in debug and yarpcmeta.
func IntrospectProcedures(routerProcs []transport.Procedure) []Procedure {
	procedures := make([]Procedure, 0, len(routerProcs))
	for _, p := range routerProcs {
		procedures = append(procedures, Procedure{
			Name:      p.Name,
			Encoding:  string(p.Encoding),
			Signature: p.Signature,
			RPCType:   p.HandlerSpec.Type().String(),
		})
	}
	return procedures
}
