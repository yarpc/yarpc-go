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

package redispubsub

import (
	"errors"
	"log"
	"sync"

	"go.uber.org/atomic"
	redis5 "gopkg.in/redis.v5"
)

var errNotStarted = errors.New("redis5client not started")

type redis5Client struct {
	addr   string
	client *redis5.Client
	pubsub *redis5.PubSub

	started atomic.Bool
	once    sync.Once
}

// NewRedis5Client creates a new Client implementation using gopkg.in/redis.v5
func NewRedis5Client(addr string) Client {
	return &redis5Client{addr: addr}
}

func (c *redis5Client) Start() error {
	c.once.Do(func() {
		c.client = redis5.NewClient(
			&redis5.Options{Addr: c.addr},
		)

		c.started.Store(true)
	})

	return c.client.Ping().Err()
}

func (c *redis5Client) Stop() error {
	if c.started.Swap(false) {
		return c.client.Close()
	}
	return nil
}

func (c *redis5Client) Publish(channel string, item []byte) error {
	if !c.started.Load() {
		return errNotStarted
	}

	cmd := c.client.Publish(channel, string(item))
	if cmd.Err() != nil {
		return errors.New("could not publish item to redis")
	}
	return nil
}

func (c *redis5Client) Subscribe(channel string, onItem func([]byte)) error {
	if !c.started.Load() {
		return errNotStarted
	}
	pubsub, err := c.client.Subscribe(channel)
	if err != nil {
		return err
	}
	c.pubsub = pubsub
	go c.readItems(onItem)
	return nil
}

func (c *redis5Client) readItems(onItem func([]byte)) {
	for {
		if !c.started.Load() {
			return
		}

		msg, err := c.pubsub.ReceiveMessage()
		if err != nil {
			log.Println("redis5 could not read subscription:", err)
		}

		onItem([]byte(msg.Payload))
	}
}
