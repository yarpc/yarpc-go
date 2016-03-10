package server

import (
	"testing"
	"testing/quick"
	"time"

	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/encoding/thrift"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestEchoRaw(t *testing.T) {
	ctx := context.Background()
	quick.Check(func(body []byte) bool {
		ctx, _ := context.WithTimeout(ctx, time.Second)
		resBody, _, err := EchoRaw(&raw.Request{
			Context:   ctx,
			Procedure: "echo/raw",
		}, body)
		assert.NoError(t, err, "")
		return assert.Equal(t, body, resBody)
	}, nil)
}

func TestEchoJSON(t *testing.T) {
	ctx := context.Background()
	quick.Check(func(body map[string][]int) bool {
		reqBody := make(map[string]interface{}, len(body))
		for k, v := range body {
			reqBody[k] = v
		}

		ctx, _ := context.WithTimeout(ctx, time.Second)
		resBody, _, err := EchoJSON(&json.Request{
			Context:   ctx,
			Procedure: "echo",
		}, reqBody)

		assert.NoError(t, err, "")
		return assert.Equal(t, reqBody, resBody)
	}, nil)
}

func TestEchoThrift(t *testing.T) {
	ctx := context.Background()
	quick.Check(func(beep string) bool {
		var e EchoThrift

		ping := &echo.Ping{Beep: beep}
		ctx, _ := context.WithTimeout(ctx, time.Second)
		pong, _, err := e.Echo(&thrift.Request{Context: ctx}, ping)

		assert.NoError(t, err, "")
		return assert.Equal(t, pong.Boop, ping.Beep)
	}, nil)
}
