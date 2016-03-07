package main

import (
	"github.com/yarpc/yarpc-go/crossdock/client"
	"github.com/yarpc/yarpc-go/crossdock/server"
)

func main() {
	server.Start()
	client.Start()
}
