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

package crossdock

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
)

// BehaviorParam is the url param representing the test to run
const BehaviorParam = "behavior"

// Dispatcher is a func that runs when the Crossdock client receives a request
type Dispatcher func(t T)

// Start begins a blocking Crossdock client
func Start(dispatcher Dispatcher) {
	http.Handle("/", requestHandler{dispatcher: dispatcher})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type requestHandler struct {
	dispatcher Dispatcher
}

func (h requestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "HEAD" {
		return
	}

	entries := Run(func(t T) {
		params := extractParams(r.Form)
		t.SetParams(params)
		h.dispatcher(t)
	})

	enc := json.NewEncoder(w)
	if err := enc.Encode(entries); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// extractParams returns a map of params from url values
func extractParams(p url.Values) (params map[string]string) {
	for k, l := range p {
		for _, v := range l {
			params[k] = v
		}
	}
	return params
}
