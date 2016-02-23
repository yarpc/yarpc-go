package main

import (
	"net/http"

	"github.com/yarpc/yarpc-go/crossdock/client"
	"github.com/yarpc/yarpc-go/crossdock/server"
)

func main() {
	server.StartServerUnderTest()
	http.HandleFunc("/", client.TestCaseHandler)
	http.ListenAndServe(":8080", nil)
}
