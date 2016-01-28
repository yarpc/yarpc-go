package main

import (
	"net/http"

	"github.com/yarpc/yarpc-go/xlang"
)

func main() {
	http.HandleFunc("/", xlang.TestCaseHandler)
	http.ListenAndServe(":8080", nil)
}
