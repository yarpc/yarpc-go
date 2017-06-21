// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package example

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	cherami_client "github.com/uber/cherami-client-go/client/cherami"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/x/cherami"
	"go.uber.org/yarpc/transport/x/cherami/example/thrift/example/exampleserviceclient"
)

// TestCheramiYARPC will create a yarpc client (using cherami transport), and issue an one way rpc call to the server
// To run this example, the cherami server(https://github.com/uber/cherami-server) needs to be started locally first
// And the destination(/test/dest) and consumer group(/test/dest_cg) needs to be created using cherami-cli
func TestCheramiYARPC(t *testing.T) {
	assert := assert.New(t)

	destination := `/test/dest`
	consumerGroup := `/test/dest_cg`

	// First let's start the server
	server := NewService(ServerConfig{
		destination:   destination,
		consumerGroup: consumerGroup,
	})

	err := server.Start()
	assert.NoError(err)
	defer server.Stop()

	// using frontend ip and port to create the cherami client is only needed in local testing
	// for a real production server, NewHyperbahnClient() should be used
	frontend := `127.0.0.1`
	port := 4922
	cheramiClient, err := cherami_client.NewClient(`example`, frontend, port, nil)
	assert.NoError(err)

	transport := cherami.NewTransport(cheramiClient)
	err = transport.Start()
	assert.NoError(err)

	// Client side needs to start the client dispatcher
	client := yarpc.NewDispatcher(yarpc.Config{
		Name: "client",
		Outbounds: yarpc.Outbounds{
			"server": {
				Oneway: transport.NewOutbound(cherami.OutboundOptions{
					Destination: destination,
				}),
			},
		},
	})
	err = client.Start()
	assert.NoError(err)
	defer client.Stop()

	// Now client side can issue the one way call
	c := exampleserviceclient.New(client.ClientConfig("server"))
	token := randomString(10)
	ack, err := c.Award(context.Background(), &token, yarpc.WithShardKey(`aa`))
	assert.NoError(err)
	fmt.Println(`received ack: `, ack)

	// Make sure server gets the call
	receivedToken := <-serverCalled
	assert.Equal(token, receivedToken)
}
