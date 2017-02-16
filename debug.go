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

package yarpc

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"sync"

	"go.uber.org/yarpc/internal/introspection"
)

var (
	dispatchersLock sync.RWMutex
	dispatchers     []*Dispatcher
)

func addDispatcherToDebugPages(disp *Dispatcher) {
	dispatchersLock.Lock()
	defer dispatchersLock.Unlock()

	dispatchers = append(dispatchers, disp)
}

func removeDispatcherFromDebugPages(disp *Dispatcher) {
	dispatchersLock.Lock()
	defer dispatchersLock.Unlock()

	for i, x := range dispatchers {
		if x == disp {
			copy(dispatchers[i:], dispatchers[i+1:])
			dispatchers[len(dispatchers)-1] = nil
			dispatchers = dispatchers[:len(dispatchers)-1]
			break
		}
	}
}

func init() {
	http.HandleFunc("/debug/yarpc", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		render(w, req)
	})
}

func render(w io.Writer, req *http.Request) {
	var data struct {
		Dispatchers []introspection.DispatcherStatus
	}

	for _, disp := range dispatchers {
		data.Dispatchers = append(data.Dispatchers, disp.Introspect())
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
			font-family: "Courier New", Courier, monospace;
		}
		table {
			color:#333333;
			border-width: 1px;
			border-color: #3A3A3A;
			border-collapse: collapse;
		}
		table th {
			border-width: 1px;
			padding: 8px;
			border-style: solid;
			border-color: #3A3A3A;
			background-color: #B3B3B3;
		}
		table td {
			border-width: 1px;
			padding: 8px;
			border-style: solid;
			border-color: #3A3A3A;
			background-color: #ffffff;
		}
	</style>
	</head>
	<body>

<h1>/debug/yarpc</h1>

{{range .Dispatchers}}
	<hr />
	<h2>Dispatcher "{{.Name}}" <small>({{.ID}})</small></h2>
	<table>
		<tr>
			<th>Procedure</th>
			<th>Encoding</th>
			<th>Signature</th>
			<th>RPC Type</th>
		</tr>
		{{range .Procedures}}
		<tr>
			<td>{{.Name}}</td>
			<td>{{.Encoding}}</td>
			<td>{{.Signature}}</td>
			<td>{{.RPCType}}</td>
		</tr>
		{{end}}
	</table>
	<h3>Inbounds</h3>
	<table>
		<tr>
			<th>Transport</th>
			<th>Endpoint</th>
			<th>State</th>
		</tr>
		{{range .Inbounds}}
		<tr>
			<td>{{.Transport}}</td>
			<td>{{.Endpoint}}</td>
			<td>{{.State}}</td>
		</tr>
		{{end}}
	</table>
	<h3>Outbounds</h3>
	<table>
		<thead>
		<tr>
			<th>Outbound Key</th>
			<th>Service</th>
			<th>Transport</th>
			<th>RPC Type</th>
			<th>Endpoint</th>
			<th>State</th>
			<th colspan="3">Chooser</th>
		</tr>
		<tr>
			<th></th>
			<th></th>
			<th></th>
			<th></th>
			<th></th>
			<th>Name</th>
			<th>State</th>
			<th>Peers</th>
		</tr>
		</thead>
		<tbody>
		{{range .Outbounds}}
		<tr>
			<td>{{.OutboundKey}}</td>
			<td>{{.Service}}</td>
			<td>{{.Transport}}</td>
			<td>{{.RPCType}}</td>
			<td>{{.Endpoint}}</td>
			<td>{{.State}}</td>
			<td>{{.Chooser.Name}}</td>
			<td>{{.Chooser.State}}</td>
			<td>
				<ul>
				{{range .Chooser.Peers}}
					<li>{{.Identifier}} ({{.State}})</li>
				{{end}}
				</ul>
			</td>
		</tr>
		</tbody>
		{{end}}
	</table>
{{end}}

	</body>
</html>
`
