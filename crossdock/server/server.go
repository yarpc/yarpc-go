package server

import (
	"fmt"
	"log"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/uber/tchannel-go"
)

// Start starts the test server that clients will make requests to
func Start() {
	ch, err := tchannel.NewChannel("yarpc-test", nil)
	if err != nil {
		log.Fatalln("couldn't create tchannel: %v", err)
	}

	rpc := yarpc.New(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: []transport.Inbound{
			http.NewInbound(":8081"),
			tch.NewInbound(ch, tch.ListenAddr(":8082")),
		},
	})

	Register(rpc)

	if err := rpc.Start(); err != nil {
		fmt.Println("error:", err.Error())
	}
}
