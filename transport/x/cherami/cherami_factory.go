// Copyright (c) 2016 Uber Technologies, Inc.
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

package cherami

import (
	"os"
	"time"

	"github.com/uber/cherami-client-go/client/cherami"
)

const (
	cheramiClientName = `cherami_yarpc`
)

// CheramiFactory provides all the interfaces that are used to get cherami entities
type CheramiFactory interface {

	// GetClientWithHyperbahn returns a cherami client using hyperbahn
	GetClientWithHyperbahn() (cherami.Client, error)

	// GetClientWithFrontEnd returns a cherami client that connects to a specific ip and port
	GetClientWithFrontEnd(ip string, port int) (cherami.Client, error)

	// GetPublisher returns a cherami publisher
	GetPublisher(client cherami.Client, destination string) (cherami.Publisher, error)

	// GetConsumer returns a cherami consumer
	GetConsumer(client cherami.Client, destination string, consumerGroup string, prefetchCount int, timeoutInSec int) (cherami.Consumer, chan cherami.Delivery, error)
}

type cheramiFactoryImp struct {
}

func NewCheramiFactory() CheramiFactory {
	return &cheramiFactoryImp{}
}

func (c *cheramiFactoryImp) GetClientWithHyperbahn() (cherami.Client, error) {
	return cherami.NewHyperbahnClient(cheramiClientName, `/etc/uber/hyperbahn/hosts.json`, nil)
}

func (c *cheramiFactoryImp) GetClientWithFrontEnd(ip string, port int) (cherami.Client, error) {
	return cherami.NewClient(cheramiClientName, ip, port, nil)
}

func (c *cheramiFactoryImp) GetPublisher(client cherami.Client, destination string) (cherami.Publisher, error) {
	publisher := client.CreatePublisher(&cherami.CreatePublisherRequest{
		Path: destination,
	})
	if err := publisher.Open(); err != nil {
		return nil, err
	}
	return publisher, nil
}

func (c *cheramiFactoryImp) GetConsumer(client cherami.Client, destination string, consumerGroup string, prefetchCount int, timeoutInSec int) (cherami.Consumer, chan cherami.Delivery, error) {
	hostName, _ := os.Hostname()
	consumerName := "yarpc_cherami_" + hostName

	consumer := client.CreateConsumer(&cherami.CreateConsumerRequest{
		Path:              destination,
		ConsumerGroupName: consumerGroup,
		ConsumerName:      consumerName,
		PrefetchCount:     prefetchCount,
		Options: &cherami.ClientOptions{
			Timeout: (time.Duration(timeoutInSec) * time.Second),
		},
	})

	ch := make(chan cherami.Delivery, prefetchCount)
	if _, err := consumer.Open(ch); err != nil {
		return nil, nil, err
	}
	return consumer, ch, nil
}
