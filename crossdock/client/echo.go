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

package client

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo/yarpc/echoclient"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
)

// echoEntry is an entry emitted by the echo behaviors.
type echoEntry struct {
	Entry

	Transport string `json:"transport"`
	Encoding  string `json:"encoding"`
	Server    string `json:"server"`
}

type echoSink struct {
	Sink

	Transport string
	Encoding  string
	Server    string
}

func (s echoSink) Put(e interface{}) {
	s.Sink.Put(echoEntry{
		Entry:     e.(Entry),
		Transport: s.Transport,
		Encoding:  s.Encoding,
		Server:    s.Server,
	})
}

// createRPC creates an RPC from the given parameters or fails the whole
// behavior.
func createRPC(s Sink, p Params) yarpc.RPC {
	server := p.Param("server")
	if server == "" {
		Fatalf(s, "server is required")
	}

	var outbound transport.Outbound
	trans := p.Param("transport")
	switch trans {
	case "http":
		outbound = http.NewOutbound(fmt.Sprintf("http://%s:8081", server))
	case "tchannel":
		ch, err := tchannel.NewChannel("yarpc-test", nil)
		if err != nil {
			Fatalf(s, "couldn't create tchannel: %v", err)
		}
		outbound = tch.NewOutbound(ch, tch.HostPort(server+":8082"))
	default:
		Fatalf(s, "unknown transport %q", trans)
	}

	return yarpc.New(yarpc.Config{
		Name:      "client",
		Outbounds: transport.Outbounds{"yarpc-test": outbound},
	})
}

// createEchoSink wraps a Sink to have transport, encoding, and server
// information.
func createEchoSink(encoding string, s Sink, p Params) Sink {
	return echoSink{
		Sink:      s,
		Transport: p.Param("transport"),
		Encoding:  encoding,
		Server:    p.Param("server"),
	}
}

// EchoRaw implements the 'raw' behavior.
func EchoRaw(s Sink, p Params) {
	s = createEchoSink("raw", s, p)
	rpc := createRPC(s, p)

	client := raw.New(rpc.Channel("yarpc-test"))
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	token := randBytes(5)
	resBody, _, err := client.Call(&raw.Request{
		Context:   ctx,
		Procedure: "echo/raw",
	}, token)

	if err != nil {
		Fatalf(s, "call to echo/raw failed: %v", err)
	}

	if !bytes.Equal(token, resBody) {
		Fatalf(s, "expected %v, got %v", token, resBody)
	}

	Successf(s, "server said: %v", resBody)
}

// jsonEcho contains an echo request or response for the JSON echo endpoint.
type jsonEcho struct {
	Token string `json:"token"`
}

// EchoJSON implements the 'json' behavior.
func EchoJSON(s Sink, p Params) {
	s = createEchoSink("json", s, p)
	rpc := createRPC(s, p)

	client := json.New(rpc.Channel("yarpc-test"))
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	var response jsonEcho
	token := randString(5)
	_, err := client.Call(
		&json.Request{Context: ctx, Procedure: "echo"},
		&jsonEcho{Token: token},
		&response,
	)

	if err != nil {
		Fatalf(s, "call to echo failed: %v", err)
	}

	if response.Token != token {
		Fatalf(s, "expected %v, got %v", token, response.Token)
	}

	Successf(s, "server said: %v", response.Token)
}

// EchoThrift implements the 'thrift' behavior.
func EchoThrift(s Sink, p Params) {
	s = createEchoSink("thrift", s, p)
	rpc := createRPC(s, p)

	client := echoclient.New(rpc.Channel("yarpc-test"))
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	token := randString(5)
	pong, _, err := client.Echo(
		&thrift.Request{Context: ctx},
		&echo.Ping{Beep: token},
	)

	if err != nil {
		Fatalf(s, "call to Echo::echo failed: %v", err)
	}

	if token != pong.Boop {
		Fatalf(s, "expected %v, got %v", token, pong.Boop)
	}

	Successf(s, "server said: %v", pong.Boop)
}

func randBytes(length int) []byte {
	out := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, out); err != nil {
		panic(err)
	}
	return out
}

func randString(length int64) string {
	bs, err := ioutil.ReadAll(io.LimitReader(rand.Reader, length))
	if err != nil {
		panic(err)
	}
	return base64.RawStdEncoding.EncodeToString(bs)
}
