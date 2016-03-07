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
