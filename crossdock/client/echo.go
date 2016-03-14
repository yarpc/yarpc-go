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
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"
	"github.com/yarpc/yarpc-go/transport/http"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
)

// echoEntry is an entry emitted by the Echo behavior.
type echoEntry struct {
	BasicEntry

	Transport string `json:"transport"`
	Encoding  string `json:"encoding"`
	Server    string `json:"server"`
}

type echoEntryBuilder struct {
	Transport string
	Encoding  string
	Server    string
}

func (b echoEntryBuilder) Skip(reason string) interface{} {
	return echoEntry{
		BasicEntry: BasicEntryBuilder.Skip(reason).(BasicEntry),
		Transport:  b.Transport,
		Encoding:   b.Encoding,
		Server:     b.Server,
	}
}

func (b echoEntryBuilder) Fail(message string) interface{} {
	return echoEntry{
		BasicEntry: BasicEntryBuilder.Fail(message).(BasicEntry),
		Transport:  b.Transport,
		Encoding:   b.Encoding,
		Server:     b.Server,
	}
}

func (b echoEntryBuilder) Pass(output string) interface{} {
	return echoEntry{
		BasicEntry: BasicEntryBuilder.Pass(output).(BasicEntry),
		Transport:  b.Transport,
		Encoding:   b.Encoding,
		Server:     b.Server,
	}
}

func runEchoBehavior(bt *BehaviorTester, encoding string) {
	server := bt.Param("server")
	trans := bt.Param("transport")
	rpc := createRPC(bt.NewBehavior(BasicEntryBuilder), server, trans)
	if rpc == nil {
		return // already logged a failure with the Behavior.
	}

	// Echo-specific Behavior used to actually make the call.
	b := bt.NewBehavior(echoEntryBuilder{
		Transport: trans,
		Encoding:  encoding,
		Server:    server,
	})
	switch encoding {
	case "raw":
		EchoRaw(b, rpc)
	case "json":
		EchoJSON(b, rpc)
	case "thrift":
		EchoThrift(b, rpc)
	default:
		b.Failf("unknown encoding %q", encoding)
	}
}

func createRPC(b Behavior, server, trans string) yarpc.RPC {
	if server == "" {
		b.Fail("server is required")
		return nil
	}

	var outbound transport.Outbound
	switch trans {
	case "http":
		outbound = http.NewOutbound(fmt.Sprintf("http://%s:8081", server))
	case "tchannel":
		ch, err := tchannel.NewChannel("yarpc-test", nil)
		if err != nil {
			b.Failf("couldn't create tchannel: %v", err)
			return nil
		}
		outbound = tch.NewOutbound(ch, tch.HostPort(server+":8082"))
	default:
		b.Failf("unknown transport %q", trans)
		return nil
	}

	return yarpc.New(yarpc.Config{
		Name:      "client",
		Outbounds: transport.Outbounds{"yarpc-test": outbound},
	})
}

// EchoRaw implements the 'raw' behavior.
func EchoRaw(b Behavior, rpc yarpc.RPC) {
	client := raw.New(rpc.Channel("yarpc-test"))
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	token := randBytes(5)
	resBody, _, err := client.Call(&raw.Request{
		Context:   ctx,
		Procedure: "echo/raw",
	}, token)

	if err != nil {
		b.Failf("call to echo/raw failed: %v", err)
		return
	}

	if !bytes.Equal(token, resBody) {
		b.Failf("expected %v, got %v", token, resBody)
		return
	}

	b.Passf("server said: %v", resBody)
}

// jsonEcho contains an echo request or response for the JSON echo endpoint.
type jsonEcho struct {
	Token string `json:"token"`
}

// EchoJSON implements the 'json' behavior.
func EchoJSON(b Behavior, rpc yarpc.RPC) {
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
		b.Failf("call to echo failed: %v", err)
		return
	}

	if response.Token != token {
		b.Failf("expected %v, got %v", token, response.Token)
		return
	}

	b.Passf("server said: %v", response.Token)
}

// EchoThrift implements the 'thrift' behavior.
func EchoThrift(b Behavior, rpc yarpc.RPC) {
	client := echo.NewEchoClient(rpc.Channel("yarpc-test"))
	ctx, _ := context.WithTimeout(context.Background(), time.Second)

	token := randString(5)
	pong, _, err := client.Echo(
		&thrift.Request{Context: ctx},
		&echo.Ping{Beep: token},
	)

	if err != nil {
		b.Failf("call to Echo::echo failed: %v", err)
		return
	}

	if token != pong.Boop {
		b.Failf("expected %v, got %v", token, pong.Boop)
		return
	}

	b.Passf("server said: %v", pong.Boop)
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
