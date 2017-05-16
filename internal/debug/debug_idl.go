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
	"fmt"
	"net/http"
	"sort"
	"strings"

	"go.uber.org/yarpc/internal/introspection"
)

type dispatcherWithIdlTree struct {
	Name    string
	ID      string
	IDLTree introspection.IDLTree
}

type idlTreeHelper struct {
	DispatcherName string
	Indent         int
	Tree           *introspection.IDLTree
}

func wrapIDLTree(dname string, i int, t *introspection.IDLTree) idlTreeHelper {
	sort.Sort(t.Modules)
	return idlTreeHelper{dname, i + 1, t}
}

type dispatcherWithIDLFile struct {
	Dispatcher introspection.DispatcherStatus
	IDLFile    introspection.IDLFile
}

const idlPagePath = "/debug/yarpc/idl/"

var idlPage = page{
	path: idlPagePath,
	handler: func(w http.ResponseWriter, req *http.Request, insp IntrospectionProvider) interface{} {
		path := strings.TrimPrefix(req.URL.Path, idlPagePath)

		// Without an IDL file paht we return an html page with a full tree.
		if path == "" {
			data := struct {
				Dispatchers     []dispatcherWithIdlTree
				PackageVersions []introspection.PackageVersion
			}{
				PackageVersions: insp.PackageVersions(),
			}

			for _, d := range insp.Dispatchers() {
				idltree := d.Procedures.IDLTree()
				idltree.Compact()
				data.Dispatchers = append(data.Dispatchers, dispatcherWithIdlTree{
					Name:    d.Name,
					ID:      d.ID,
					IDLTree: idltree,
				})
			}

			return data
		}

		// Here we are hunting for the idl file path to return. Dispatchers can have
		// duplicated names. Because
		parts := strings.SplitN(path, "/", 2)
		var selectDispatcher string
		var selectIDL string
		if path != "" {
			if len(parts) != 2 {
				w.WriteHeader(400)
				fmt.Fprintf(w, "Invalid arguments")
				return nil
			}
			selectDispatcher = parts[0]
			selectIDL = parts[1]
		}

		dispatchers := insp.DispatchersByName(selectDispatcher)

		if len(dispatchers) == 0 {
			w.WriteHeader(404)
			fmt.Fprintf(w, "dispatcher(s) %q not found", selectDispatcher)
			return nil
		}

		var idls []dispatcherWithIDLFile

		for _, d := range dispatchers {
			if m, ok := d.Procedures.IDLFileByFilePath(selectIDL); ok {
				idls = append(idls, dispatcherWithIDLFile{
					Dispatcher: d,
					IDLFile:    *m,
				})
			}
		}

		if len(idls) == 0 {
			w.WriteHeader(404)
			fmt.Fprintf(w, "IDL %q not found on Dispatcher(s) %q\n",
				selectIDL, selectDispatcher)
			return nil
		}

		w.Header().Set("Content-Type", "text/plain")
		for _, d := range idls {
			if len(idls) > 1 {
				fmt.Fprintf(w, "Dispatcher %q (%q):\n",
					d.Dispatcher.Name, d.Dispatcher.ID)
			}
			fmt.Fprintf(w, d.IDLFile.Content)
		}
		return nil
	},
	html: `
{{ define "title"}}/debug/yarpc/idl{{ end }}
{{ define "body" }}
{{range .Dispatchers}}
	<hr />
	<h2>Dispatcher "{{.Name}}" <small>({{.ID}})</small></h2>
	<table class="tree">
		<tr>
			<th>File</th>
			<th>Checksum</th>
			<th>Includes</th>
		</tr>
		{{ template "idltree" (wrapIDLtree .Name -1 .IDLTree) }}
	</div>
{{end}}
{{end}}
{{ define "idltree" }}
{{ $dname := .DispatcherName }}
{{ $indent := .Indent }}
{{ with .Tree }}
	{{range .Modules}}
		<tr>
			<td style="padding-left: {{ $indent }}em">
				<div class="filename" id="{{$dname}}/{{.FilePath}}">
					<a href="{{$dname}}/{{.FilePath}}">{{pathBase .FilePath}}</a>
				<div>
			</td>
			<td class="sha1">
				{{ .Checksum }}
			</td>
			<td class="includes">
				{{ range .Includes }}
					<a class="anchor" href="#{{$dname}}/{{.FilePath}}">{{pathBase .FilePath}}</a>
				{{ end }}
			</td>
		</tr>
	{{end}}
	{{range $dir, $subTree := .Dir}}
		<tr>
			<td style="padding-left: {{ $indent }}em">
				<div class="filename">
					{{ $dir }}/
				</div>
			</td>
		</tr>
		{{ template "idltree" (wrapIDLtree $dname $indent $subTree) }}
	{{end}}
</tr>
{{ end }}
{{ end }}
`,
}
