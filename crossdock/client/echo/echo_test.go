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

package echo

import (
	"testing"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/client/behavior"
	"github.com/yarpc/yarpc-go/crossdock/server"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo/yarpc/echoserver"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/tchannel-go"
)

type invalidThriftEcho struct{}

func (invalidThriftEcho) Echo(req *thrift.Request, ping *echo.Ping) (*echo.Pong, *thrift.Response, error) {
	return &echo.Pong{Boop: "derp"}, nil, nil
}

func TestEchoBehaviors(t *testing.T) {
	transports := []struct {
		name       string
		getInbound func() transport.Inbound
	}{
		{
			"http",
			func() transport.Inbound {
				return http.NewInbound(":8081")
			},
		},
		{
			"tchannel",
			func() transport.Inbound {
				ch, err := tchannel.NewChannel("yarpc-test", nil)
				require.NoError(t, err, "could not create TChannel")
				return tch.NewInbound(ch, tch.ListenAddr(":8082"))
			},
		},
	}

	tests := []struct {
		encname  string
		register func(yarpc.RPC)
		// registerInvalid func(yarpc.RPC)
		behavior func(behavior.Sink, behavior.Params)

		// expectations: one of these must be set
		passed  string
		skipped string
		failed  string
	}{
		{
			encname:  "raw",
			behavior: Raw,
			failed:   `unknown procedure "echo/raw"`,
		},
		{
			encname:  "raw",
			behavior: Raw,
			register: func(rpc yarpc.RPC) {
				raw.Register(rpc, raw.Procedure("echo/raw",
					func(req *raw.Request, body []byte) ([]byte, *raw.Response, error) {
						return []byte("lol"), nil, nil
					}))
			},
			failed: "got [108 111 108]",
		},
		{
			encname:  "raw",
			behavior: Raw,
			register: func(rpc yarpc.RPC) {
				raw.Register(rpc, raw.Procedure("echo/raw", server.EchoRaw))
			},
			passed: "server said:",
		},
		{
			encname:  "json",
			behavior: JSON,
			failed:   `unknown procedure "echo"`,
		},
		{
			encname:  "json",
			behavior: JSON,
			register: func(rpc yarpc.RPC) {
				json.Register(rpc, json.Procedure("echo",
					func(req *json.Request, body map[string]interface{}) (
						map[string]interface{}, *json.Response, error) {
						return map[string]interface{}{"token": "invalid"}, nil, nil
					}))
			},
			failed: "got invalid",
		},
		{
			encname:  "json",
			behavior: JSON,
			register: func(rpc yarpc.RPC) {
				json.Register(rpc, json.Procedure("echo", server.EchoJSON))
			},
			passed: "server said:",
		},
		{
			encname:  "thrift",
			behavior: Thrift,
			failed:   `unknown procedure "Echo::echo"`,
		},
		{
			encname:  "thrift",
			behavior: Thrift,
			register: func(rpc yarpc.RPC) {
				thrift.Register(rpc, echoserver.New(invalidThriftEcho{}))
			},
			failed: "got derp",
		},
		{
			encname:  "thrift",
			behavior: Thrift,
			register: func(rpc yarpc.RPC) {
				thrift.Register(rpc, echoserver.New(server.EchoThrift{}))
			},
			passed: "server said:",
		},
	}

	for _, trans := range transports {
		for _, tt := range tests {
			params := behavior.ParamsFromMap{
				"transport": trans.name,
				"server":    "localhost",
			}

			rpc := yarpc.New(yarpc.Config{
				Name:     "yarpc-test",
				Inbounds: []transport.Inbound{trans.getInbound()},
			})

			if tt.register != nil {
				tt.register(rpc)
			}

			entries := behavior.Run(func(s behavior.Sink) {
				err := rpc.Start()
				assert.NoError(t, err,
					"failed to start RPC for %v, %v", trans.name, tt.encname)
				if err == nil {
					defer rpc.Stop()
					tt.behavior(s, params)
				}
			})

			if !assert.Len(t, entries, 1) {
				continue
			}

			e := entries[0].(echoEntry)
			assert.Equal(t, trans.name, e.Transport)
			assert.Equal(t, "localhost", e.Server)
			assert.Equal(t, tt.encname, e.Encoding)

			var status behavior.Status
			var message string
			if tt.failed != "" {
				status = behavior.Failed
				message = tt.failed
			} else if tt.skipped != "" {
				status = behavior.Skipped
				message = tt.skipped
			} else if tt.passed != "" {
				status = behavior.Passed
				message = tt.passed
			} else {
				panic("one of failed, skipped, and success must be set")
			}

			assert.Equal(t, status, e.Status, "status mismatch for %v, %v", trans.name, tt.encname)
			assert.Contains(t, e.Output, message, "output mismatch for %v, %v", trans.name, tt.encname)
		}
	}
}
