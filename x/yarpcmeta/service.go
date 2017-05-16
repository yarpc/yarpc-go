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
	"errors"

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
	procedures.BasicIDLOnly()
	return &procsResponse{
		Service:    m.disp.Name(),
		Procedures: procedures,
	}, nil
}

func (m *service) introspect(ctx context.Context, body interface{}) (*introspection.DispatcherStatus, error) {
	status := m.disp.Introspect()
	status.Procedures.BasicIDLOnly()
	return &status, nil
}

type idlsQuery struct {
	EntryPoint string   `json:"entryPoint,omitempty"`
	Selection  []string `json:"selection,omitempty"`
}

type idlFile struct {
	FilePath string   `json:"filePath"`
	Checksum string   `json:"checksum"`
	Includes []string `json:"includes,omitempty"`
	Content  string   `json:"content,omitempty"`
}

type idlsResponse struct {
	IDLs []idlFile `json:"idls"`
}

func (r *idlsResponse) Append(m *introspection.IDLFile) bool {
	includes := make([]string, len(m.Includes))
	for i := range m.Includes {
		includes[i] = m.Includes[i].FilePath
	}
	r.IDLs = append(r.IDLs, idlFile{
		FilePath: m.FilePath,
		Checksum: m.Checksum,
		Includes: includes,
		Content:  m.Content,
	})
	return false
}

func (m *service) idls(ctx context.Context, rq *idlsQuery) (*idlsResponse, error) {
	procedures := introspection.IntrospectProcedures(m.disp.Router().Procedures())

	// return the flattened tree by default.
	if rq == nil || (rq.EntryPoint == "" && len(rq.Selection) == 0) {
		var r idlsResponse
		procedures.Flatmap(r.Append)
		return &r, nil
	}

	if rq.EntryPoint != "" && len(rq.Selection) > 0 {
		return nil, errors.New(`ask for either an "entryPoint" or a "selection", but not both`)
	}

	// find the idl requested among all of them and return it with all its
	// recursive includes.
	if rq.EntryPoint != "" {
		var r idlsResponse
		procedures.Flatmap(func(m *introspection.IDLFile) bool {
			if m.FilePath == rq.EntryPoint {
				m.Flatmap(r.Append)
				return true
			}
			return false
		})
		return &r, nil
	}

	// find and return every idl matching the selection. No includes are returned.
	var selection map[string]struct{}
	if len(rq.Selection) > 0 {
		selection = make(map[string]struct{})
		for _, s := range rq.Selection {
			selection[s] = struct{}{}
		}
	}

	var r idlsResponse
	procedures.Flatmap(func(m *introspection.IDLFile) bool {
		if _, ok := selection[m.FilePath]; ok {
			r.Append(m)
		}
		return false
	})

	return &r, nil
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
		{"yarpc::idls", m.idls,
			`idls() {...}`},
	}
	var r []transport.Procedure
	for _, m := range methods {
		p := json.Procedure(m.Name, m.Handler)[0]
		p.Signature = m.Signature
		r = append(r, p)
	}
	return r
}
