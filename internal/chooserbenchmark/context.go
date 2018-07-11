// Copyright (c) 2018 Uber Technologies, Inc.
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

package chooserbenchmark

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"go.uber.org/multierr"
)

// Context is an objects bundle contains all information for benchmark
// will be passed among different modules in system across the whole lifecycle
type Context struct {
	// server context
	ServerCount int
	Listeners   Listeners
	Servers     []*Server

	// client context
	ClientCount int
	Clients     []*Client

	// clients and servers synchronization
	WG          sync.WaitGroup
	ServerStart chan struct{}
	ClientStart chan struct{}
	ServerStop  chan struct{}
	ClientStop  chan struct{}

	// other configurations
	Duration time.Duration
	Writer   io.Writer
}

func (ctx *Context) buildServers(config *Config) error {
	ctx.Servers = make([]*Server, len(ctx.Listeners))
	id := 0
	for _, group := range config.ServerGroups {
		for i := 0; i < group.Count; i++ {
			lis, err := ctx.Listeners.Listener(id)
			if err != nil {
				return err
			}
			sigma := DefaultLogNormalSigma
			if group.LogNormalSigma != 0 {
				// when sigma is 0, we use default value 0.5, if you want value 0, use Epsilon
				sigma = group.LogNormalSigma
			}
			server, err := NewServer(id, group.Name, group.Latency, sigma, lis, ctx.ServerStart, ctx.ServerStop, &ctx.WG)
			if err != nil {
				return err
			}
			ctx.Servers[id] = server
			id++
		}
	}
	return nil
}

func (ctx *Context) buildClients(config *Config, clientCount int) error {
	ctx.Clients = make([]*Client, clientCount)
	start := time.Now()
	var wg sync.WaitGroup
	total := 0
	for _, group := range config.ClientGroups {
		total += group.Count
	}
	wg.Add(total)
	// time complexity for start all clients is O(ServerCount*ClientCount),
	// each client has its own peer list so this could be parallel
	id := 0
	for _, group := range config.ClientGroups {
		for j := 0; j < group.Count; j++ {
			client := NewClient(id, &group, ctx.Listeners, ctx.ClientStart, ctx.ClientStop, &ctx.WG)
			ctx.Clients[id] = client
			go func() {
				// Start will append all peers to list, so it's O(ServerCount) time complexity
				if err := client.chooser.Start(); err != nil {
					log.Fatal(err)
				}
				wg.Done()
			}()
			id++
		}
	}
	wg.Wait()
	end := time.Now()
	fmt.Fprintf(ctx.Writer, "build %d clients with %d servers in %v\n", total, len(ctx.Listeners), end.Sub(start))
	return nil
}

// NewContext returns a Context object based on input configuration
func NewContext(config *Config) (*Context, error) {
	if config.Output == nil {
		config.Output = os.Stdout
	}
	ctx := Context{
		Duration:    config.Duration,
		ServerStart: make(chan struct{}),
		ClientStart: make(chan struct{}),
		ServerStop:  make(chan struct{}),
		ClientStop:  make(chan struct{}),
		Writer:      config.Output,
	}

	serverCount, clientCount := 0, 0
	for _, group := range config.ServerGroups {
		serverCount += group.Count
	}
	for _, group := range config.ClientGroups {
		clientCount += group.Count
	}

	ctx.Listeners = NewListeners(serverCount)

	if err := multierr.Combine(ctx.buildServers(config), ctx.buildClients(config, clientCount)); err != nil {
		return nil, err
	}

	return &ctx, nil
}

// Launch contains the main work flow, start clients and servers, run benchmark,
// collect metrics and visualize them.
func (ctx *Context) Launch() error {
	serverCount := len(ctx.Servers)
	clientCount := len(ctx.Clients)

	fmt.Fprintf(ctx.Writer, "launch %d servers...\n", serverCount)
	for _, server := range ctx.Servers {
		ctx.WG.Add(1)
		go server.Serve()
	}
	close(ctx.ServerStart)
	// wait until all servers start, ensure all servers are ready when clients
	// begin to issue requests
	ctx.WG.Wait()

	fmt.Fprintf(ctx.Writer, "launch %d clients...\n", clientCount)
	for _, client := range ctx.Clients {
		go client.Start()
	}

	fmt.Fprintf(ctx.Writer, "begin benchmark, over after %d seconds...\n", ctx.Duration/time.Second)
	close(ctx.ClientStart)
	time.Sleep(ctx.Duration)

	// wait until all servers stop
	ctx.WG.Add(clientCount)
	close(ctx.ClientStop)
	ctx.WG.Wait()
	// wait until all clients stop
	ctx.WG.Add(serverCount)
	close(ctx.ServerStop)
	ctx.WG.Wait()

	return nil
}

// Visualize do the visualization of metrics data stored in context
func (ctx *Context) Visualize() error {
	fmt.Fprintf(ctx.Writer, "\nbenchmark end, collect metrics and visualize...")

	vis, err := NewVisualizer(ctx)
	if err != nil {
		return err
	}

	for _, groupName := range vis.clientGroupNames {
		meta := vis.clientData[groupName]
		meta.visualizeClientGroup(vis, ctx.Writer)
	}

	for _, groupName := range vis.serverGroupNames {
		meta := vis.serverData[groupName]
		meta.visualizeServerGroup(vis, ctx.Writer)
	}
	return nil
}
