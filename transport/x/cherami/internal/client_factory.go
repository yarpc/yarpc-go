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

package internal

import (
	"os"
	"time"

	"github.com/uber/cherami-client-go/client/cherami"
)

// ClientFactory provides all the interfaces that are used to get cherami entities
type ClientFactory interface {

	// GetClientWithHyperbahn returns a cherami client using hyperbahn
	GetClientWithHyperbahn(serviceName string, hostFile string) (cherami.Client, error)

	// GetClientWithFrontEnd returns a cherami client that connects to a specific ip and port
	GetClientWithFrontEnd(serviceName string, ip string, port int) (cherami.Client, error)

	// GetPublisher returns a cherami publisher
	GetPublisher(client cherami.Client, destination string) (cherami.Publisher, error)

	// GetConsumer returns a cherami consumer
	GetConsumer(client cherami.Client, config ConsumerConfig) (cherami.Consumer, chan cherami.Delivery, error)
}

// ConsumerConfig is the configuration needed to create a consumer object
type ConsumerConfig struct {
	Destination   string
	ConsumerGroup string
	PrefetchCount int
	Timeout       time.Duration
}

type clientFactoryImp struct {
}

// NewClientFactory creates a client factory object
func NewClientFactory() ClientFactory {
	return &clientFactoryImp{}
}

func (c *clientFactoryImp) GetClientWithHyperbahn(serviceName string, hostFile string) (cherami.Client, error) {
	return cherami.NewHyperbahnClient(serviceName, hostFile, nil)
}

func (c *clientFactoryImp) GetClientWithFrontEnd(serviceName string, ip string, port int) (cherami.Client, error) {
	return cherami.NewClient(serviceName, ip, port, nil)
}

func (c *clientFactoryImp) GetPublisher(client cherami.Client, destination string) (cherami.Publisher, error) {
	publisher := client.CreatePublisher(&cherami.CreatePublisherRequest{
		Path: destination,
	})
	err := publisher.Open()
	return publisher, err
}

func (c *clientFactoryImp) GetConsumer(client cherami.Client, config ConsumerConfig) (cherami.Consumer, chan cherami.Delivery, error) {
	hostName, _ := os.Hostname()
	consumerName := "yarpc_cherami_" + hostName

	consumer := client.CreateConsumer(&cherami.CreateConsumerRequest{
		Path:              config.Destination,
		ConsumerGroupName: config.ConsumerGroup,
		ConsumerName:      consumerName,
		PrefetchCount:     config.PrefetchCount,
		Options: &cherami.ClientOptions{
			Timeout: config.Timeout,
		},
	})

	ch := make(chan cherami.Delivery, config.PrefetchCount)
	if _, err := consumer.Open(ch); err != nil {
		return nil, nil, err
	}
	return consumer, ch, nil
}
