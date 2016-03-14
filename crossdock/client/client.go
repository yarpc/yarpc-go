package client

import (
	"encoding/json"
	"net/http"
)

// Start begins a blocking Crossdock client
func Start() {
	http.HandleFunc("/", behaviorRequestHandler)
	http.ListenAndServe(":8080", nil)
}

func behaviorRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "HEAD" {
		return
	}

	// All behaviors will eventually contribute to this behavior's output.
	bt := BehaviorTester{Params: httpParams{Request: r}}
	v := bt.Param("behavior")
	switch v {
	case "raw", "json", "thrift":
		runEchoBehavior(&bt, v)
	default:
		bt.NewBehavior(BasicEntryBuilder).Skipf("unknown behavior %v", v)
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(bt.Entries); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// httpParams provides access to behavior parameters that are stored inside an
// HTTP request.
type httpParams struct {
	Request *http.Request
}

func (h httpParams) Param(name string) string {
	return h.Request.FormValue(name)
}
