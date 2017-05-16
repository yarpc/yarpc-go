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

package introspection

import (
	"go.uber.org/yarpc/api/transport"
)

// IntrospectProcedures is a convenience function that translate a slice of
// transport.Procedure to a slice of introspection.Procedure. This output is
// used in debug and yarpcmeta.
func IntrospectProcedures(routerProcs []transport.Procedure) Procedures {
	procedures := make([]Procedure, 0, len(routerProcs))
	for _, p := range routerProcs {
		var spec interface{}
		switch p.HandlerSpec.Type() {
		case transport.Unary:
			spec = p.HandlerSpec.Unary()
		case transport.Oneway:
			spec = p.HandlerSpec.Oneway()
		}
		var IDLEntryPoint *IDLFile
		if spec != nil {
			if i, ok := spec.(IntrospectableHandler); ok {
				if i := i.Introspect(); i != nil {
					IDLEntryPoint = i.IDLEntryPoint
				}
			}
		}
		procedures = append(procedures, Procedure{
			Name:          p.Name,
			Encoding:      string(p.Encoding),
			Signature:     p.Signature,
			RPCType:       p.HandlerSpec.Type().String(),
			IDLEntryPoint: IDLEntryPoint,
		})
	}
	return procedures
}

// Procedure represent a registered procedure on a dispatcher.
type Procedure struct {
	Name          string   `json:"name"`
	Encoding      string   `json:"encoding"`
	Signature     string   `json:"signature"`
	RPCType       string   `json:"rpcType"`
	IDLEntryPoint *IDLFile `json:"idlEntryPoint,omitempty"`
}

// Procedures is a slice of Procedure.
type Procedures []Procedure

// Flatmap iterates recursively over every idl modules. The cb function is
// called onces for every unique idl module (based on FilePath). Iteration is
// aborted if cb returns true.
func (ps Procedures) Flatmap(cb func(*IDLFile) bool) {
	ft := newIDLFlattener(cb)
	for _, p := range ps {
		if p.IDLEntryPoint != nil {
			ft.Collect(p.IDLEntryPoint)
		}
	}
}

// IDLFiles returns a slice with all idl modules.
func (ps Procedures) IDLFiles() IDLFiles {
	var r []IDLFile
	ps.Flatmap(func(m *IDLFile) bool {
		r = append(r, *m)
		return false
	})
	return r
}

// IDLFileByFilePath returns the idl modules matching the given filepath.
func (ps Procedures) IDLFileByFilePath(filePath string) (r *IDLFile, ok bool) {
	ps.Flatmap(func(m *IDLFile) bool {
		if m.FilePath == filePath {
			r = m
			return true
		}
		return false
	})
	return r, r != nil
}

// IDLTree builds an IDL tree from all idl modules.
func (ps Procedures) IDLTree() IDLTree {
	tb := newIDLTreeBuilder()
	for _, p := range ps {
		if p.IDLEntryPoint != nil {
			tb.Collect(p.IDLEntryPoint)
		}
	}
	return tb.Root
}

// BasicIDLOnly filters out the content and references to included IDLs on
// every procedures.
func (ps Procedures) BasicIDLOnly() {
	for _, p := range ps {
		if p.IDLEntryPoint != nil {
			p.IDLEntryPoint.Content = ""
			p.IDLEntryPoint.Includes = nil
		}
	}
}
