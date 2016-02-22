package client

import (
	"fmt"
	"net/http"
)

// TestCaseHandler drives the test suite as instructed by Crossdock
func TestCaseHandler(w http.ResponseWriter, r *http.Request) {
	behavior := r.FormValue("behavior")
	server := r.FormValue("server")

	if behavior == "" || server == "" {
		fmt.Fprint(w, "handler is ready, please send behavior and server params")
		return
	}

	switch behavior {
	case "echo":
		PrintResult(w, EchoBehavior(server))
		return
	}
}

// Result contains the result of a behavior's execution
type Result struct {
	Passed  bool
	Message string
}

// PrintResult writes tap to w for a given Result
func PrintResult(w http.ResponseWriter, result Result) {
	tap := "not ok"
	if result.Passed == true {
		tap = "ok"
	}
	message := fmt.Sprintf("%v - %v", tap, result.Message)
	fmt.Fprint(w, message)
}
