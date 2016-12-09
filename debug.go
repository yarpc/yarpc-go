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

package yarpc

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"

	"go.uber.org/yarpc/internal/debug"

	"golang.org/x/net/trace"
)

var (
	dispatchers []Dispatcher
)

func (d *dispatcher) Debug() debug.Dispatcher {
	var inbounds []debug.Inbound
	for _, i := range d.inbounds {
		inbounds = append(inbounds, i.Debug())
	}
	return debug.Dispatcher{
		Name:       d.Name,
		ID:         fmt.Sprintf("%p", d),
		Procedures: d.DebugProcedures(),
		Inbounds:   inbounds,
	}
}

func init() {
	http.HandleFunc("/debug/yarpc", func(w http.ResponseWriter, req *http.Request) {
		any, sensitive := trace.AuthRequest(req)
		if !any {
			http.Error(w, "not allowed", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		Render(w, req, sensitive)
	})
}

func AddDebugPagesFor(disp Dispatcher) {
	dispatchers = append(dispatchers, disp)
}

func Render(w io.Writer, req *http.Request, sensitive bool) {
	var data struct {
		Dispatchers []debug.Dispatcher
	}

	for _, disp := range dispatchers {
		data.Dispatchers = append(data.Dispatchers, disp.Debug())
	}

	if err := pageTmpl.ExecuteTemplate(w, "Page", data); err != nil {
		log.Printf("yarpc/debug: Failed executing template: %v", err)
	}
}

var pageTmpl = template.Must(template.New("Page").Funcs(template.FuncMap{}).Parse(pageHTML))

const pageHTML = `
<html>
	<head>
	<title>/debug/yarpc</title>
	<style type="text/css">
		body {
			font-family: sans-serif;
		}
		table {
			text-align: left;
		}
	</style>
	</head>
	<body>

<h1>/debug/yarpc</h1>

{{range .Dispatchers}}
	<hr />
	<h3>Dispatcher "{{.Name}}" <small>({{.ID}})</small></h3>
	<table>
		<tr>
			<th>Service</th>
			<th>Procedure</th>
			<th>RPC flavor</th>
			<th>Encoding</th>
			<th>Signature</th>
		</tr>
		{{range .Procedures}}
		<tr>
			<td class="service">{{.Service}}</td>
			<td class="procname">{{.Name}}</td>
			<td class="rpcflavor">{{.Flavor}}</td>
			<td class="encoding">{{.Encoding}}</td>
			<td class="signature">{{.Signature}}</td>
		</tr>
		{{end}}
	</table>
	<h4>Inbound</h4>
	<table>
		<tr>
			<th>Transport</th>
			<th>Endpoint</th>
			<th>Peer</th>
			<th>State</th>
		</tr>
		{{range .Inbounds}}
		<tr>
			<td>{{.Transport}}</td>
			<td>{{.Endpoint}}</td>
			<td>{{.Peer}}</td>
			<td>{{.State}}</td>
		</tr>
		{{end}}
	</table>
	<h4>Outbound</h4>
	<table>
		<tr>
			<th>Name</th>
			<th>Transport</th>
			<th>Endpoint</th>
			<th>Peer</th>
			<th>State</th>
		</tr>
		{{range .Outbounds}}
		<tr>
			<td>{{.Name}}</td>
			<td>{{.Transport}}</td>
			<td>{{.Endpoint}}</td>
			<td>{{.Peer}}</td>
			<td>{{.State}}</td>
		</tr>
		{{end}}
	</table>
{{end}}

	</body>
</html>
`
