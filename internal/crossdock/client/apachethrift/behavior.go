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

package apachethrift

import (
	"fmt"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/thrift"
	"go.uber.org/yarpc/internal/crossdock/client/gauntlet"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/peer/single"
	"go.uber.org/yarpc/transport/http"

	"github.com/crossdock/crossdock-go"
)

const (
	serverParam = "apachethriftserver"
	serverPort  = 8088
)

// Run runs the apachethrift behavior
func Run(t crossdock.T) {
	fatals := crossdock.Fatals(t)

	server := t.Param(serverParam)
	fatals.NotEmpty(server, "apachethriftserver is required")

	httpTransport := http.NewTransport()
	hostPort := hostport.PeerIdentifier(fmt.Sprintf("%v:%v", server, serverPort))

	thriftOutbound := http.NewChooserOutbound(single.New(hostPort, httpTransport)).
		WithURLTemplate("http://host:port/thrift/ThriftTest")
	secondOutbound := http.NewChooserOutbound(single.New(hostPort, httpTransport)).
		WithURLTemplate("http://host:port/thrift/SecondService")
	multiplexOutbound := http.NewChooserOutbound(single.New(hostPort, httpTransport)).
		WithURLTemplate("http://host:port/thrift/multiplexed")

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: "apache-thrift-client",
		Outbounds: yarpc.Outbounds{
			"ThriftTest": {
				Unary:  thriftOutbound,
				Oneway: thriftOutbound,
			},
			"SecondService": {
				Unary: secondOutbound,
			},
			"Multiplexed": {
				Unary:  multiplexOutbound,
				Oneway: multiplexOutbound,
			},
		},
	})
	fatals.NoError(dispatcher.Start(), "could not start Dispatcher")
	defer dispatcher.Stop()

	// We can just run all the gauntlet tests against each URL because
	// tests for undefined methods are skipped.
	tests := []struct {
		ServerName string
		Services   gauntlet.ServiceSet
		Options    []thrift.ClientOption
	}{
		{
			ServerName: "ThriftTest",
			Services:   gauntlet.ThriftTest,
		},
		{
			ServerName: "SecondService",
			Services:   gauntlet.SecondService,
		},
		{
			ServerName: "Multiplexed",
			Services:   gauntlet.AllServices,
			Options:    []thrift.ClientOption{thrift.Multiplexed},
		},
	}

	for _, tt := range tests {
		t.Tag("outbound", tt.ServerName)
		gauntlet.RunGauntlet(t, gauntlet.Config{
			Dispatcher:    dispatcher,
			ServerName:    tt.ServerName,
			Envelope:      true,
			Services:      tt.Services,
			ClientOptions: tt.Options,
		})
	}
}
