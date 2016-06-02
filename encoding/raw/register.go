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

import "github.com/yarpc/yarpc-go/transport"

// Registrant is used for types that define or know about different Raw
// procedures.
type Registrant interface {
	// Gets a mapping from procedure name to the Handler for that procedure
	// for all procedures provided by the registrant.
	getHandlers() map[string]Handler
}

// Handler implements a single procedure.
type Handler func(*ReqMeta, []byte) ([]byte, *ResMeta, error)

// procedure is a registrant with a single handler.
type procedure struct {
	Name    string
	Handler Handler
}

func (p procedure) getHandlers() map[string]Handler {
	return map[string]Handler{p.Name: p.Handler}
}

// Procedure builds a Registrant with a single procedure in it.
func Procedure(name string, handler Handler) Registrant {
	return procedure{Name: name, Handler: handler}
}

// Register registers the procedures defined by the given Registrant with the
// given Registry.
func Register(reg transport.Registry, registrant Registrant) {
	for name, handler := range registrant.getHandlers() {
		reg.Register("", name, rawHandler{handler})
	}
}
