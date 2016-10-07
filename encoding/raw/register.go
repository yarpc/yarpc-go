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

package raw

import (
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport"

	"golang.org/x/net/context"
)

// Register calls the Registry's Register method.
//
// This function exists for backwards compatibility only. It will be removed
// in a future version.
//
// Deprecated: Use the Registry's Register method directly.
func Register(r transport.Registry, rs []transport.Registrant) {
	r.Register(rs)
}

// Handler implements a single procedure.
type Handler func(context.Context, yarpc.ReqMeta, []byte) ([]byte, yarpc.ResMeta, error)

// Procedure builds a Registrant from the given raw handler.
func Procedure(name string, handler Handler) []transport.Registrant {
	return []transport.Registrant{
		{
			Procedure: name,
			Handler:   rawHandler{handler},
		},
	}
}
