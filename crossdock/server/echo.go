package server

import "github.com/yarpc/yarpc-go/encoding/json"

// EchoRequest contains a message to echo
type EchoRequest struct {
	Token string `json:"token"`
}

// EchoResponse contains a messaged echoed by EchoRequest
type EchoResponse struct {
	Token string `json:"token"`
}

// Echo echoes an EchoResponse
func Echo(req *json.Request, body *EchoRequest) (*EchoResponse, *json.Response, error) {
	return &EchoResponse{Token: body.Token}, nil, nil
}
