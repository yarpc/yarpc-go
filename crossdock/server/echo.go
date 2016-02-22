package server

import (
	"github.com/yarpc/yarpc-go"
	"golang.org/x/net/context"
)

// EchoRequest contains a message to echo
type EchoRequest struct {
	Token string `json:"token"`
}

// EchoResponse contains a messaged echoed by EchoRequest
type EchoResponse struct {
	Token string `json:"token"`
}

// Echo echoes an EchoResponse
func Echo(ctx context.Context, meta yarpc.Meta, req *EchoRequest) (*EchoResponse, yarpc.Meta, error) {
	return &EchoResponse{Token: req.Token}, nil, nil
}
