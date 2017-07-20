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

package debug

import (
	"html/template"
	"log"
	"net/http"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/internal/introspection"
)

var (
	// DefaultTmpl is the default template used.
	DefaultTmpl = template.Must(template.New("tmpl").Parse(`
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
		header::after {
			content: "";
			clear: both;
			display: table;
		}
		h1 {
			width: 40%;
			float: left;
			margin: 0;
		}
		div.dependencies {
			width: 60%;
			float: left;
			font-size: small;
			text-align: right;
		}
	</style>
	</head>
	<body>

<header>
<h1>/debug/yarpc</h1>
<div class="dependencies">
	{{range .PackageVersions}}
	<span>{{.Name}}={{.Version}}</span>
	{{end}}
</div>
</header>

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
`))

	// DefaultLogFunc is the default logging function used when there is a rendering error.
	DefaultLogFunc = log.Printf
)

// HandlerOption is an option for a new Handler.
type HandlerOption func(*handler)

// WithTemplate sets the template to be used for rendering.
//
// By default, DefaultTmpl is used.
func WithTemplate(tmpl *template.Template) HandlerOption {
	return func(handler *handler) {
		handler.tmpl = tmpl
	}
}

// WithLogFunc set the logging function to use when there is a rendering error.
//
// If set to nil, no logging will be performed.
//
// By default, log.Printf is ued.
func WithLogFunc(logFunc func(string, ...interface{})) HandlerOption {
	return func(handler *handler) {
		handler.logFunc = logFunc
	}
}

// NewHandler returns a http.HandlerFunc to expose dispatcher status and package versions.
func NewHandler(dispatcher *yarpc.Dispatcher, opts ...HandlerOption) http.HandlerFunc {
	return newHandler(dispatcher, opts...).handle
}

type handler struct {
	dispatcher *yarpc.Dispatcher
	tmpl       *template.Template
	logFunc    func(string, ...interface{})
}

func newHandler(dispatcher *yarpc.Dispatcher, opts ...HandlerOption) *handler {
	handler := &handler{
		dispatcher: dispatcher,
		tmpl:       DefaultTmpl,
		logFunc:    DefaultLogFunc,
	}
	for _, opt := range opts {
		opt(handler)
	}
	return handler
}

func (h *handler) handle(responseWriter http.ResponseWriter, request *http.Request) {
	data := struct {
		Dispatchers     []introspection.DispatcherStatus
		PackageVersions []introspection.PackageVersion
	}{
		Dispatchers: []introspection.DispatcherStatus{
			h.dispatcher.Introspect(),
		},
		PackageVersions: yarpc.PackageVersions,
	}

	responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.Execute(responseWriter, data); err != nil {
		if h.logFunc != nil {
			h.logFunc("yarpc/debug: failed executing template: %v", err)
		}
	}
}
