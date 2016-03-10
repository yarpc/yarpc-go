package server

import (
	"github.com/yarpc/yarpc-go/crossdock/thrift/echo"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/encoding/raw"
	"github.com/yarpc/yarpc-go/encoding/thrift"
	"github.com/yarpc/yarpc-go/transport"
)

// Register the different endpoints of the TestSubject with the given
// Registry.
func Register(reg transport.Registry) {
	raw.Register(reg, raw.Procedure("echo/raw", EchoRaw))
	json.Register(reg, json.Procedure("echo", EchoJSON))
	thrift.Register(reg, echo.NewEchoHandler(EchoThrift{}))
}

// EchoRaw implements the echo/raw procedure.
func EchoRaw(req *raw.Request, body []byte) ([]byte, *raw.Response, error) {
	return body, nil, nil
}

// EchoJSON implements the echo procedure.
func EchoJSON(req *json.Request, body map[string]interface{}) (map[string]interface{}, *json.Response, error) {
	return body, nil, nil
}

// EchoThrift implements the Thrift Echo service.
type EchoThrift struct{}

// Echo endpoint for the Echo service.
func (EchoThrift) Echo(req *thrift.Request, ping *echo.Ping) (*echo.Pong, *thrift.Response, error) {
	return &echo.Pong{Boop: ping.Beep}, nil, nil
}
