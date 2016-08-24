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
	"net/url"
	"testing"

	"go.uber.org/yarpc/crossdock/client"
	"go.uber.org/yarpc/crossdock/server"

	"github.com/crossdock/crossdock-go"
)

const clientURL = "http://127.0.0.1:8080"

func TestCrossdock(t *testing.T) {
	server.Start()
	defer server.Stop()
	go client.Start()

	crossdock.Wait(t, clientURL, 10)

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
			name: "errorstchout",
		},
		{
			name: "headers",
			axes: axes{
				"transport": []string{"http", "tchannel"},
				"encoding":  []string{"raw", "json", "thrift"},
			},
		},
		{
			name: "tchclient",
			axes: axes{
				"encoding": []string{"raw", "json", "thrift"},
			},
		},
		{
			name: "tchserver",
			axes: axes{
				"encoding": []string{"raw", "json", "thrift"},
			},
		},
		{
			name: "thriftgauntlet",
			axes: axes{
				"transport": []string{"http", "tchannel"},
			},
		},
		{
			name: "outboundttl",
			axes: axes{
				"transport": []string{"http", "tchannel"},
			},
		},
		{
			name: "ctxpropagation",
			axes: axes{
				"transport": []string{"http", "tchannel"},
			},
			params: params{
				"ctxserver": "127.0.0.1",
				"ctxclient": "127.0.0.1",
			},
		},
	}

	for _, bb := range behaviors {

		args := url.Values{}
		for k, v := range defaultParams {
			args.Set(k, v)
		}

		if len(bb.axes) == 0 {
			crossdock.Call(t, clientURL, bb.name, args)
			continue
		}

		for _, entry := range crossdock.Combinations(bb.axes) {
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

			crossdock.Call(t, clientURL, bb.name, entryArgs)
		}
	}
}
