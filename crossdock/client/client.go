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

package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Start begins a blocking Crossdock client
func Start() {
	http.HandleFunc("/", testCaseHandler)
	http.ListenAndServe(":8080", nil)
}

func testCaseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "HEAD" {
		return
	}
	behavior := r.FormValue("behavior")
	server := r.FormValue("server")
	switch behavior {
	case "echo":
		fmt.Fprintf(w, respond(EchoBehavior(server)))
	default:
		res, _ := json.Marshal(response{{Status: skipped, Output: "Not implemented"}})
		fmt.Fprintf(w, string(res))
	}
}

const passed = "passed"
const skipped = "skipped"
const failed = "failed"

type response []subResponse
type subResponse struct {
	Status string `json:"status"`
	Output string `json:"output"`
}

func respond(output string, err error) string {
	if err != nil {
		s, _ := json.Marshal(response{{Status: failed, Output: err.Error()}})
		return string(s)
	}
	s, _ := json.Marshal(response{{Status: passed, Output: output}})
	return string(s)
}
