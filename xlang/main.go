package main

import "net/http"

func main() {
	http.HandleFunc("/", TestCaseHandler)
	http.ListenAndServe(":8080", nil)
}
