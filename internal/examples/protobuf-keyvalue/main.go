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

package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"

	"go.uber.org/yarpc/internal/examples/protobuf-keyvalue/kv"
)

var (
	outboundName = flag.String("outbound", "tchannel", "name of the outbound to use (http/tchannel)")

	errRequestNil    = errors.New("request nil")
	errRequestKeyNil = errors.New("request key nil")
)

func main() {
	if err := do(); err != nil {
		log.Fatal(err)
	}
}

func do() error {
	flag.Parse()
	if err := startServer(); err != nil {
		return err
	}
	return doClient()
}

func startServer() error {
	tchannelTransport, err := tchannel.NewChannelTransport(
		tchannel.ServiceName("kv"),
		tchannel.ListenAddr(":28941"),
	)
	if err != nil {
		return err
	}
	dispatcher := yarpc.NewDispatcher(
		yarpc.Config{
			Name: "kv",
			Inbounds: yarpc.Inbounds{
				tchannelTransport.NewInbound(),
				http.NewTransport().NewInbound(":24034"),
			},
		},
	)
	dispatcher.Register(kv.BuildAPIProcedures(newAPIServer()))
	return dispatcher.Start()
}

func doClient() error {
	outbound, err := getOutbound()
	if err != nil {
		return err
	}
	dispatcher := yarpc.NewDispatcher(
		yarpc.Config{
			Name: "kv-client",
			Outbounds: yarpc.Outbounds{
				"kv": {
					Unary: outbound,
				},
			},
		},
	)
	if err := dispatcher.Start(); err != nil {
		return err
	}
	defer dispatcher.Stop()

	apiClient := kv.NewAPIClient(dispatcher.ClientConfig("kv"))
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		args := strings.Split(line, " ")
		if len(args) < 1 || len(args[0]) < 3 {
			continue
		}

		cmd := args[0]
		args = args[1:]
		switch cmd {
		case "get":
			if len(args) != 1 {
				fmt.Println("usage: get key")
				continue
			}
			key := args[0]
			ctx, cancel := newContextWithTimeout()
			defer cancel()
			if response, err := apiClient.GetValue(ctx, &kv.GetValueRequest{key}); err != nil {
				fmt.Printf("get %s failed: %s\n", key, err.Error())
			} else {
				fmt.Println(key, "=", response.Value)
			}
			continue
		case "set":
			if len(args) != 1 && len(args) != 2 {
				fmt.Println("usage: set key [value]")
				continue
			}
			key := args[0]
			value := ""
			if len(args) == 2 {
				value = args[1]
			}

			ctx, cancel := newContextWithTimeout()
			defer cancel()
			if _, err := apiClient.SetValue(ctx, &kv.SetValueRequest{key, value}); err != nil {
				fmt.Printf("set %s = %s failed: %v\n", key, value, err.Error())
			}
			continue
		case "exit":
			return nil
		default:
			fmt.Println("invalid command", cmd)
			fmt.Println("valid commands are: get, set, exit")
		}
	}
	return scanner.Err()
}

func getOutbound() (transport.UnaryOutbound, error) {
	switch strings.ToLower(*outboundName) {
	case "http":
		return http.NewTransport().NewSingleOutbound("http://127.0.0.1:24034"), nil
	case "tchannel":
		tchannelTransport, err := tchannel.NewChannelTransport(tchannel.ServiceName("kv-client"))
		if err != nil {
			return nil, err
		}
		return tchannelTransport.NewSingleOutbound("localhost:28941"), nil
	default:
		return nil, fmt.Errorf("invalid outbound: %s", *outboundName)
	}
}

func newContextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 1*time.Second)
}

type apiServer struct {
	sync.RWMutex
	items map[string]string
}

func newAPIServer() *apiServer {
	return &apiServer{sync.RWMutex{}, make(map[string]string)}
}

func (a *apiServer) GetValue(ctx context.Context, request *kv.GetValueRequest) (*kv.GetValueResponse, error) {
	if request == nil {
		return nil, errRequestNil
	}
	if request.Key == "" {
		return nil, errRequestKeyNil
	}
	a.RLock()
	if value, ok := a.items[request.Key]; ok {
		a.RUnlock()
		return &kv.GetValueResponse{value}, nil
	}
	a.RUnlock()
	return nil, fmt.Errorf("key not set: %s", request.Key)
}

func (a *apiServer) SetValue(ctx context.Context, request *kv.SetValueRequest) (*kv.SetValueResponse, error) {
	if request == nil {
		return nil, errRequestNil
	}
	if request.Key == "" {
		return nil, errRequestKeyNil
	}
	a.Lock()
	if request.Value == "" {
		delete(a.items, request.Key)
	} else {
		a.items[request.Key] = request.Value
	}
	a.Unlock()
	return nil, nil
}
