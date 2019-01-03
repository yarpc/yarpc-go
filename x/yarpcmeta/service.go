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

package yarpcmeta

import (
	"context"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/internal/introspection"
)

// Register new yarpc meta procedures a dispatcher, exposing information about
// the dispatcher itself.
func Register(d *yarpc.Dispatcher) {
	ms := &service{d}
	d.Register(ms.Procedures())
}

// service exposes dispatcher informations via Procedures().
type service struct {
	disp *yarpc.Dispatcher
}

type procsResponse struct {
	Service    string                    `json:"service"`
	Procedures []introspection.Procedure `json:"procedures"`
}

func (m *service) procs(ctx context.Context, body interface{}) (*procsResponse, error) {
	procedures := introspection.IntrospectProcedures(m.disp.Router().Procedures())
	return &procsResponse{
		Service:    m.disp.Name(),
		Procedures: procedures,
	}, nil
}

func (m *service) introspect(ctx context.Context, body interface{}) (*introspection.DispatcherStatus, error) {
	status := m.disp.Introspect()
	return &status, nil
}

// Procedures returns the procedures to register on a dispatcher.
func (m *service) Procedures() []transport.Procedure {
	methods := []struct {
		Name      string
		Handler   interface{}
		Signature string
	}{
		{"yarpc::procedures", m.procs,
			`procedures() {"service": "...", "procedures": [{"name": "..."}]}`},
		{"yarpc::introspect", m.introspect,
			`introspect() {...}`},
	}
	var r []transport.Procedure
	for _, m := range methods {
		p := json.Procedure(m.Name, m.Handler)[0]
		p.Signature = m.Signature
		r = append(r, p)
	}
	return r
}
