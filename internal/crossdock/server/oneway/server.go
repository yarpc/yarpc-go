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

package oneway

import (
	"log"
	"os"
	"strings"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/internal/crossdock/thrift/oneway/onewayserver"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/x/redis"
	"go.uber.org/yarpc/transport/x/cherami"

	 cherami_client "github.com/uber/cherami-client-go/client/cherami"
	 cherami_type "github.com/uber/cherami-thrift/.generated/go/cherami"
)

var dispatcher *yarpc.Dispatcher

// Start starts the test server that clients will make requests to
func Start() {
	httpTransport := http.NewTransport()
	inbounds := []transport.Inbound{httpTransport.NewInbound(":8084")}

	if useRedis() {
		rds := redis.NewInbound(
			redis.NewRedis5Client("redis:6379"),
			"yarpc/oneway",
			"yarpc/oneway/processing",
			time.Second,
		)
		inbounds = append(inbounds, rds)
	}

	destination := `/test/dest`
	consumerGroup := `/test/dest_cg`
	consumedRetention := int32(300)
	unconsumedRetention := int32(600)
	cheramiClient, err := cherami_client.NewClient(`example`, `cherami`, 4922, nil)
	if err != nil {
		log.Println(`error creating cherami client %v`, err)
		return
	}

	_, err = cheramiClient.CreateDestination(&cherami_type.CreateDestinationRequest{
		Path: &destination,
		ConsumedMessagesRetention:   &consumedRetention,
		UnconsumedMessagesRetention: &unconsumedRetention,
	})
	if err != nil && !strings.Contains(err.Error(), `EntityAlreadyExistsError`) {
		log.Println(`error creating destination %v`, err)
		return
	}

	_, err = cheramiClient.CreateConsumerGroup(&cherami_type.CreateConsumerGroupRequest{
		DestinationPath: &destination,
		ConsumerGroupName: &consumerGroup,
	})
	if err != nil && !strings.Contains(err.Error(), `EntityAlreadyExistsError`) {
		log.Println(`error creating consumer group %v`, err)
		return
	}

	cheramiTransport := cherami.NewTransport(cheramiClient)
	if err := cheramiTransport.Start(); err != nil {
		log.Println(`error starting cherami transport`)
		return
	}

	inbounds = append(inbounds, cheramiTransport.NewInbound(cherami.InboundConfig{
			Destination: destination,
			ConsumerGroup: consumerGroup,
		}))

	dispatcher = yarpc.NewDispatcher(yarpc.Config{
		Name:     "oneway-server",
		Inbounds: inbounds,
	})

	// Echo procedures make an RPC back to the caller with the same context,
	// and body over http/raw
	h := &onewayHandler{http.NewTransport()}
	dispatcher.Register(raw.OnewayProcedure("echo/raw", h.EchoRaw))
	dispatcher.Register(json.OnewayProcedure("echo/json", h.EchoJSON))
	dispatcher.Register(onewayserver.New(h))

	if err := dispatcher.Start(); err != nil {
		log.Println("oneway server dispatcher failed to load:", err.Error())
	}
}

// Stop stops running the RPC test subject
func Stop() {
	if dispatcher == nil {
		return
	}
	if err := dispatcher.Stop(); err != nil {
		log.Println("oneway server dispatcher failed to stop:", err.Error())
	}
}

// useRedis checks to see if a redis server is expected to be
// available
func useRedis() bool {
	return os.Getenv("REDIS") == "enabled"
}
