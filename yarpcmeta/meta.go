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

package yarpcmeta

import (
	"context"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/json"
)

// Service exposes dispatcher informations via Procedures().
type Service struct {
	disp *yarpc.Dispatcher
}

// NewMetaService allocates a new yarpc meta service, exposing the list of
// services and registered procedures of the dispatcher via Procedures().
func NewMetaService(d *yarpc.Dispatcher) *Service {
	return &Service{d}
}

// Register allocates a new yarpc meta service from the dispatcher and
// registers it right away on it.
func Register(d *yarpc.Dispatcher) *Service {
	ms := NewMetaService(d)
	d.Register(ms.Procedures())
	return ms
}

type procedure struct {
	Service   string
	Name      string
	Encoding  string
	Signature string
	RPCType   string
}

type procsResponse struct {
	Name     string
	Services map[string][]procedure
}

func (m *Service) procs(ctx context.Context, body interface{}) (*procsResponse, error) {
	routerProcs := m.disp.Router().Procedures()
	services := make(map[string][]procedure)
	for _, p := range routerProcs {
		pinfo := procedure{
			Service:   p.Service,
			Name:      p.Name,
			Encoding:  string(p.Encoding),
			Signature: p.Signature,
			RPCType:   p.HandlerSpec.Type().String(),
		}
		services[p.Service] = append(services[p.Service], pinfo)
	}
	return &procsResponse{
		Name:     m.disp.Name(),
		Services: services,
	}, nil
}

// Procedures returns the procedures to register on a dispatcher.
func (m *Service) Procedures() []transport.Procedure {
	methods := []struct {
		Name      string
		Handler   interface{}
		Signature string
	}{
		{"procedures", m.procs, `procs() {"Name": "...", "Services": {"...": [{"Name": "..."}]}}`},
	}
	var r []transport.Procedure
	for _, m := range methods {
		p := json.Procedure(m.Name, m.Handler)[0]
		p.Service = "yarpc"
		p.Signature = m.Signature
		r = append(r, p)
	}
	return r
}
