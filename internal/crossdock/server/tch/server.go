// Copyright (c) 2025 Uber Technologies, Inc.
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

package tch

import (
	"github.com/uber/tchannel-go"
	"github.com/uber/tchannel-go/json"
	"github.com/uber/tchannel-go/raw"
	"github.com/uber/tchannel-go/thrift"
	"go.uber.org/yarpc/internal/crossdock/thrift/gen-go/echo"
	"go.uber.org/yarpc/internal/crossdock/thrift/gen-go/gauntlet_tchannel"
	"golang.org/x/net/context"
)

var (
	log      = tchannel.SimpleLogger
	ch       *tchannel.Channel
	hostPort = ":8083"
)

// Start starts the tch testing server
func Start() {
	ch, err := tchannel.NewChannel("tchannel-server", &tchannel.ChannelOptions{Logger: tchannel.SimpleLogger})
	if err != nil {
		log.WithFields(tchannel.ErrField(err)).Fatal("Couldn't create new channel.")
	}

	if err := register(ch); err != nil {
		log.WithFields(tchannel.ErrField(err)).Fatal("Couldn't register channel.")
	}

	if err := ch.ListenAndServe(hostPort); err != nil {
		log.WithFields(
			tchannel.LogField{Key: "hostPort", Value: hostPort},
			tchannel.ErrField(err),
		).Fatal("Couldn't listen.")
	}
}

// Stop stops the tch testing server
func Stop() {
	if ch != nil {
		ch.Close()
	}
}

// Register the different endpoints of the test subject
func register(ch *tchannel.Channel) error {
	ch.Register(raw.Wrap(echoRawHandler{}), "echo/raw")
	ch.Register(raw.Wrap(handlerTimeoutRawHandler{}), "handlertimeout/raw")

	if err := json.Register(ch, json.Handlers{"echo": echoJSONHandler}, onError); err != nil {
		return err
	}

	tserver := thrift.NewServer(ch)
	tserver.Register(echo.NewTChanEchoServer(&echoThriftHandler{}))
	tserver.Register(gauntlet_tchannel.NewTChanThriftTestServer(&thriftTestHandler{}))
	tserver.Register(gauntlet_tchannel.NewTChanSecondServiceServer(&secondServiceHandler{}))
	return nil
}

func onError(ctx context.Context, err error) {
	log.WithFields(tchannel.ErrField(err)).Fatal("onError handler triggered.")
}
