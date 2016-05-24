package yarpc

import (
	"testing"
	"testing/quick"
	"time"

	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestEchoRaw(t *testing.T) {
	ctx := context.Background()
	quick.Check(func(headers transport.Headers, body []byte) bool {
		ctx, _ := context.WithTimeout(ctx, time.Second)
		resBody, res, err := EchoRaw(&raw.Request{
			Context:   ctx,
			Headers:   headers,
			Procedure: "echo/raw",
		}, body)
		assert.NoError(t, err, "")
		return assert.Equal(t, body, resBody) && assert.Equal(t, headers, res.Headers)
	}, nil)
}

func TestEchoJSON(t *testing.T) {
	ctx := context.Background()
	quick.Check(func(headers transport.Headers, body map[string][]int) bool {
		reqBody := make(map[string]interface{}, len(body))
		for k, v := range body {
			reqBody[k] = v
		}

		ctx, _ := context.WithTimeout(ctx, time.Second)
		resBody, res, err := EchoJSON(&json.Request{
			Context:   ctx,
			Headers:   headers,
			Procedure: "echo",
		}, reqBody)

		assert.NoError(t, err, "")
		return assert.Equal(t, reqBody, resBody) && assert.Equal(t, headers, res.Headers)
	}, nil)
}

func TestEchoThrift(t *testing.T) {
	ctx := context.Background()
	quick.Check(func(headers transport.Headers, beep string) bool {
		var e EchoThrift

		ping := &echo.Ping{Beep: beep}
		ctx, _ := context.WithTimeout(ctx, time.Second)
		pong, res, err := e.Echo(&thrift.Request{
			Context: ctx,
			Headers: headers,
		}, ping)

		assert.NoError(t, err, "")
		return assert.Equal(t, pong.Boop, ping.Beep) && assert.Equal(t, headers, res.Headers)
	}, nil)
}
