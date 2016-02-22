package main

import (
	"net/http"
	"time"

	"github.com/yarpc/yarpc-go/crossdock/client"
	"github.com/yarpc/yarpc-go/crossdock/server"
)

func main() {
	// TODO need to be able to wait till all inbounds are finished listening
	go server.StartServerUnderTest()
	// TODO maybe sleep?
	time.Sleep(2 * time.Second)

	http.HandleFunc("/", client.TestCaseHandler)
	http.ListenAndServe(":8080", nil)
}
