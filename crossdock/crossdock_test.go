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

package main

import (
	"encoding/json"
	"log"
	"net/url"
	"testing"
	"time"

	"github.com/yarpc/yarpc-go/crossdock/client"
	"github.com/yarpc/yarpc-go/crossdock/server"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

const clientURL = "http://127.0.0.1:8080"

type result struct {
	Status string `json:"status"`
	Output string `json:"output"`
}

func TestCrossdock(t *testing.T) {
	server.Start()
	defer server.Stop()
	go client.Start()

	wait(t)

	type params map[string]string
	type axes map[string][]string

	defaultParams := params{"server": "127.0.0.1"}

	behaviors := []struct {
		name   string
		params params
		axes   axes
	}{
		{
			name: "raw",

			axes: axes{"transport": []string{"http", "tchannel"}},
		},
		{
			name: "json",
			axes: axes{"transport": []string{"http", "tchannel"}},
		},
		{
			name: "thrift",
			axes: axes{"transport": []string{"http", "tchannel"}},
		},
		{
			name: "errors",
		},
		{
			name: "headers",
			axes: axes{
				"transport": []string{"http", "tchannel"},
				"encoding":  []string{"raw", "json", "thrift"},
			},
		},
	}

	for _, bb := range behaviors {

		args := url.Values{}
		args.Set("behavior", bb.name)
		for k, v := range defaultParams {
			args.Set(k, v)
		}

		if len(bb.axes) == 0 {
			call(t, args)
			continue
		}

		for _, entry := range combinations(bb.axes) {
			entryArgs := url.Values{}
			for k := range args {
				entryArgs.Set(k, args.Get(k))
			}
			for k, v := range entry {
				entryArgs.Set(k, v)
			}
			for k, v := range bb.params {
				entryArgs.Set(k, v)
			}

			call(t, entryArgs)
		}
	}
}

func wait(t *testing.T) {
	attempts := 0
	ctx := context.Background()
	for {
		if attempts > 9 {
			t.Fatalf("could not talk to client in %d attempts", attempts)
		}

		attempts++

		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		log.Println("HEAD", clientURL)

		_, err := ctxhttp.Head(ctx, nil, clientURL)
		if err == nil {
			break
		}

		sleepFor := 100 * time.Millisecond
		log.Println(err, "- sleeping for", sleepFor)
		time.Sleep(sleepFor)
	}
	log.Println("Client is ready, beginning test...")
}

func call(t *testing.T, args url.Values) {
	u, err := url.Parse("http://127.0.0.1:8080")
	require.NoError(t, err, "failed to parse URL")

	u.RawQuery = args.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("GET", u.String())

	res, err := ctxhttp.Get(ctx, nil, u.String())

	require.NoError(t, err, "request %v failed", args)
	defer res.Body.Close()

	var results []result
	require.NoError(t, json.NewDecoder(res.Body).Decode(&results),
		"failed to decode response for %v", args)

	for _, result := range results {
		if result.Status != "passed" && result.Status != "skipped" {
			t.Errorf("request %v failed: %s", args, result.Output)
		}
	}
}
