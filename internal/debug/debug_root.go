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
	"net/http"

	"go.uber.org/yarpc/internal/introspection"
)

var rootPage = page{
	path: "/debug/yarpc",
	handler: func(w http.ResponseWriter, req *http.Request, insp IntrospectionProvider) interface{} {
		return struct {
			Dispatchers     []introspection.DispatcherStatus
			PackageVersions []introspection.PackageVersion
		}{
			PackageVersions: insp.PackageVersions(),
			Dispatchers:     insp.Dispatchers(),
		}
	},
	html: `
{{ define "title"}}/debug/yarpc{{ end }}
{{ define "body" }}
{{range .Dispatchers}}
	<hr />
	<h2>Dispatcher "{{.Name}}" <small>({{.ID}})</small></h2>
	<table class="spreadsheet">
		<tr>
			<th>Procedure</th>
			<th>Encoding</th>
			<th>Signature</th>
			<th>RPC Type</th>
			<th><a href="yarpc/idl">IDLs</a> Entry point</th>
		</tr>
		{{$dname := .Name}}
		{{range .Procedures}}
		<tr>
			<td>{{.Name}}</td>
			<td>{{.Encoding}}</td>
			<td>{{.Signature}}</td>
			<td>{{.RPCType}}</td>
			<td>
			{{ if .IDLEntryPoint }}
				<a href="yarpc/idl/{{$dname}}/{{ .IDLEntryPoint.FilePath }}">
				{{ .IDLEntryPoint.FilePath }}</a>
			{{ end }}
			</td>
		</tr>
		{{end}}
	</table>
	<h3>Inbounds</h3>
	<table class="spreadsheet">
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
	<table class="spreadsheet">
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
{{ end }}`,
}
