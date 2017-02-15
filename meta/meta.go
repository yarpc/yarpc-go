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

package meta

import (
	"context"
	"sort"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/json"
)

type MetaService struct {
	disp *yarpc.Dispatcher
}

// NewMetaService allocates a new meta service, exposing the list of services
// and registered procedures of the dispatcher.
func NewMetaService(d *yarpc.Dispatcher) *MetaService {
	return &MetaService{d}
}

type procedure struct {
	Service   string
	Name      string
	Encoding  string
	Signature string
	RPCType   string
}

type procsResponse struct {
	Name       string
	Services   []string
	Procedures []procedure
}

func (m *MetaService) procs(ctx context.Context, body interface{}) (*procsResponse, error) {
	routerProcs := m.disp.Router().Procedures()
	procedures := make([]procedure, 0, len(routerProcs))
	servicesMap := make(map[string]struct{})
	for _, p := range routerProcs {
		procedures = append(procedures, procedure{
			Service:   p.Service,
			Name:      p.Name,
			Encoding:  string(p.Encoding),
			Signature: p.Signature,
			RPCType:   p.HandlerSpec.Type().String(),
		})
		servicesMap[p.Service] = struct{}{}
	}
	services := make([]string, 0, len(servicesMap))
	for k := range servicesMap {
		services = append(services, k)
	}
	sort.Strings(services)
	return &procsResponse{
		Name:       m.disp.Name(),
		Services:   services,
		Procedures: procedures,
	}, nil
}

// Procedures returns the procedures to register on a dispatcher.
func (m *MetaService) Procedures() []transport.Procedure {
	methods := []struct {
		Name      string
		Handler   interface{}
		Signature string
	}{
		{"procedures", m.procs, `procs() json(registered procedures)`},
	}
	var r []transport.Procedure
	for _, m := range methods {
		p := json.Procedure(m.Name, m.Handler)[0]
		p.Service = "meta"
		p.Signature = m.Signature
		r = append(r, p)
	}
	return r
}
