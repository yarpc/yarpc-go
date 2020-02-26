// Copyright (c) 2020 Uber Technologies, Inc.
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
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/examples/protobuf/example"
	"go.uber.org/yarpc/internal/examples/protobuf/examplepb"
	"go.uber.org/yarpc/internal/examples/protobuf/exampleutil"
	"go.uber.org/yarpc/internal/testutils"
	"go.uber.org/yarpc/yarpcerrors"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
)

var (
	flagSet        = flag.NewFlagSet("protobuf", flag.ExitOnError)
	flagOutbound   = flagSet.String("outbound", "tchannel", "The outbound to use for unary calls")
	flagGoogleGRPC = flagSet.Bool("google-grpc", false, "Use google grpc for outbound KeyValue calls")
	flagBlock      = flagSet.Bool("block", false, "Block and run the server forever instead of running the client")
)

func main() {
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
	if err := do(); err != nil {
		log.Fatal(err)
	}
}

func do() error {
	transportType, err := testutils.ParseTransportType(*flagOutbound)
	if err != nil {
		return err
	}
	return run(
		transportType,
		*flagGoogleGRPC,
		*flagBlock,
		os.Stdin,
		os.Stdout,
	)
}

func run(
	transportType testutils.TransportType,
	googleGRPC bool,
	block bool,
	input io.Reader,
	output io.Writer,
) error {
	keyValueYARPCServer := example.NewKeyValueYARPCServer()
	sinkYARPCServer := example.NewSinkYARPCServer(true)
	fooYARPCServer := example.NewFooYARPCServer(transport.NewHeaders())
	logger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	return exampleutil.WithClients(
		transportType,
		keyValueYARPCServer,
		sinkYARPCServer,
		fooYARPCServer,
		logger,
		func(clients *exampleutil.Clients) error {
			return doClient(
				keyValueYARPCServer,
				sinkYARPCServer,
				clients,
				googleGRPC,
				block,
				input,
				output,
			)
		},
	)
}

func doClient(
	keyValueYARPCServer *example.KeyValueYARPCServer,
	sinkYARPCServer *example.SinkYARPCServer,
	clients *exampleutil.Clients,
	googleGRPC bool,
	block bool,
	input io.Reader,
	output io.Writer,
) error {
	if block {
		log.Println("ready")
		select {}
	}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintln(output, line)
		args := strings.Split(line, " ")
		if len(args) < 1 || len(args[0]) < 3 {
			continue
		}
		cmd := args[0]
		args = args[1:]
		switch cmd {
		case "get":
			if len(args) != 1 {
				fmt.Fprintln(output, "usage: get key")
				continue
			}
			key := args[0]
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			var response *examplepb.GetValueResponse
			var err error
			if googleGRPC {
				response, err = clients.KeyValueGRPCClient.GetValue(clients.ContextWrapper.Wrap(ctx), &examplepb.GetValueRequest{Key: key})
			} else {
				response, err = clients.KeyValueYARPCClient.GetValue(ctx, &examplepb.GetValueRequest{Key: key})
			}
			if err != nil {
				fmt.Fprintf(output, "get %s failed: %s\n", key, getErrorMessage(err))
			} else {
				fmt.Fprintln(output, key, "=", response.Value)
			}
			continue
		case "set":
			if len(args) != 1 && len(args) != 2 {
				fmt.Fprintln(output, "usage: set key [value]")
				continue
			}
			key := args[0]
			value := ""
			if len(args) == 2 {
				value = args[1]
			}
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			var err error
			if googleGRPC {
				_, err = clients.KeyValueGRPCClient.SetValue(clients.ContextWrapper.Wrap(ctx), &examplepb.SetValueRequest{Key: key, Value: value})
			} else {
				_, err = clients.KeyValueYARPCClient.SetValue(ctx, &examplepb.SetValueRequest{Key: key, Value: value})
			}
			if err != nil {
				fmt.Fprintf(output, "set %s = %s failed: %v\n", key, value, getErrorMessage(err))
			}
			continue
		case "fire":
			if len(args) != 1 {
				fmt.Fprintln(output, "usage: fire value")
				continue
			}
			value := args[0]
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			if _, err := clients.SinkYARPCClient.Fire(ctx, &examplepb.FireRequest{Value: value}); err != nil {
				fmt.Fprintf(output, "fire %s failed: %s\n", value, getErrorMessage(err))
			}
			if err := sinkYARPCServer.WaitFireDone(); err != nil {
				fmt.Fprintln(output, err)
			}
			continue
		case "fired-values":
			if len(args) != 0 {
				fmt.Fprintln(output, "usage: fired-values")
				continue
			}
			fmt.Fprintln(output, strings.Join(sinkYARPCServer.Values(), " "))
			continue
		case "exit":
			return nil
		default:
			return fmt.Errorf("invalid command, valid commands are: get, set, fire, fired-values, exit: %s", cmd)
		}
	}
	return scanner.Err()
}

func getErrorMessage(err error) string {
	if yarpcerrors.IsStatus(err) {
		return yarpcerrors.FromError(err).Message()
	}
	if status, ok := status.FromError(err); ok {
		return status.Message()
	}
	return err.Error()
}
