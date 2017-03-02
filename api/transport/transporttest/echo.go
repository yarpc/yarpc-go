package transporttest

import (
	"context"
	"io"

	"go.uber.org/yarpc/api/transport"
)

// EchoRouter is a router that echoes all unary inbound requests, with no
// explicit procedures.
type EchoRouter struct{}

// Procedures returns no explicitly supported procedures.
func (EchoRouter) Procedures() []transport.Procedure {
	return nil
}

// Choose always returns a unary echo handler.
func (EchoRouter) Choose(ctx context.Context, req *transport.Request) (transport.HandlerSpec, error) {
	return echoHandlerSpec, nil
}

// EchoHandler is a unary handler that echoes the request body on the response
// body for any inbound request.
type EchoHandler struct{}

// Handle handles an inbound request by copying the request body to the
// response body.
func (EchoHandler) Handle(ctx context.Context, req *transport.Request, resw transport.ResponseWriter) error {
	_, err := io.Copy(resw, req.Body)
	return err
}

var echoHandlerSpec = transport.NewUnaryHandlerSpec(EchoHandler{})
