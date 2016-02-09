package server

import (
	"fmt"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
)

// StartServerUnderTest starts the test server that clients will make requests to
func StartServerUnderTest() {
	yarpc := yarpc.New(yarpc.Config{
		Name: "yarpc-test",
		Inbounds: []transport.Inbound{
			http.Inbound(":8081"),
		},
	})

	json.Register(yarpc, json.Procedure("echo", Echo))

	if err := yarpc.Start(); err != nil {
		fmt.Println("error:", err.Error())
	}
}
