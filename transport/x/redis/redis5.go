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

package redis

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/atomic"
	redis5 "gopkg.in/redis.v5"
)

var errNotStarted = errors.New("redis5client not started")

type redis5Client struct {
	addr   string
	client *redis5.Client

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

// IsRunning returns whether the redis client is running.
func (c *redis5Client) IsRunning() bool {
	return c.started.Load()
}

func (c *redis5Client) LPush(queueKey string, item []byte) error {
	if !c.started.Load() {
		return errNotStarted
	}

	cmd := c.client.LPush(queueKey, item)
	if cmd.Err() != nil {
		return errors.New("could not push item onto queue")
	}
	return nil
}

func (c *redis5Client) BRPopLPush(queueKey, processingKey string, timeout time.Duration) ([]byte, error) {
	if !c.started.Load() {
		return nil, errNotStarted
	}

	cmd := c.client.BRPopLPush(queueKey, processingKey, timeout)

	item, _ := cmd.Bytes()
	// No bytes means that we timed out waiting for something in our queue
	// and we should try again
	if len(item) == 0 {
		return nil, errors.New("no item found in queue")
	}

	return item, nil
}

func (c *redis5Client) LRem(key string, item []byte) error {
	if !c.started.Load() {
		return errNotStarted
	}

	removed := c.client.LRem(key, 1, item).Val()
	if removed <= 0 {
		return errors.New("could not remove item from queue")
	}
	return nil
}

// Endpoint returns the enpoint configured for this client.
func (c *redis5Client) Endpoint() string {
	return c.addr
}

// ConState returns the status of the connection(s).
func (c *redis5Client) ConState() string {
	ps := c.client.PoolStats()
	active := ps.TotalConns - ps.FreeConns
	return fmt.Sprintf("%d/%d connection(s)", active, ps.TotalConns)
}
