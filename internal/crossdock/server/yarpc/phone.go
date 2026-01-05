// Copyright (c) 2026 Uber Technologies, Inc.
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

package yarpc

import (
	"context"
	js "encoding/json"
	"fmt"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/transport/http"
	"go.uber.org/yarpc/transport/tchannel"
)

// HTTPTransport contains information about an HTTP transport.
type HTTPTransport struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// TChannelTransport contains information about a TChannel transport.
type TChannelTransport struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// TransportConfig contains the transport configuration for the phone request.
type TransportConfig struct {
	HTTP     *HTTPTransport     `json:"http"`
	TChannel *TChannelTransport `json:"tchannel"`
}

// PhoneRequest is a request to make another request to a different service.
type PhoneRequest struct {
	Service   string          `json:"service"`
	Procedure string          `json:"procedure"`
	Transport TransportConfig `json:"transport"`
	Body      *js.RawMessage  `json:"body"`
}

// PhoneResponse is the response of a Phone request.
type PhoneResponse struct {
	Service   string         `json:"service"`
	Procedure string         `json:"procedure"`
	Body      *js.RawMessage `json:"body"`
}

// Phone implements the phone procedure
func Phone(ctx context.Context, body *PhoneRequest) (*PhoneResponse, error) {
	var outbound transport.UnaryOutbound

	httpTransport := http.NewTransport()
	tchannelTransport, err := tchannel.NewChannelTransport(tchannel.ServiceName("yarpc-test-client"))
	if err != nil {
		return nil, fmt.Errorf("failed to build ChannelTransport: %v", err)
	}

	switch {
	case body.Transport.HTTP != nil:
		t := body.Transport.HTTP
		outbound = httpTransport.NewSingleOutbound(fmt.Sprintf("http://%s:%d", t.Host, t.Port))
	case body.Transport.TChannel != nil:
		t := body.Transport.TChannel
		hostport := fmt.Sprintf("%s:%d", t.Host, t.Port)
		outbound = tchannelTransport.NewSingleOutbound(hostport)
	default:
		return nil, fmt.Errorf("unconfigured transport")
	}

	if err := outbound.Start(); err != nil {
		return nil, err
	}
	defer outbound.Stop()

	// TODO use yarpc.Service for caller
	client := json.New(clientconfig.MultiOutbound("yarpc-test", body.Service, transport.Outbounds{
		Unary: outbound,
	}))
	resBody := PhoneResponse{
		Service:   "yarpc-test", // TODO use yarpc.Service
		Procedure: yarpc.CallFromContext(ctx).Procedure(),
	}

	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	if err := client.Call(ctx, body.Procedure, body.Body, &resBody.Body); err != nil {
		return nil, err
	}

	return &resBody, nil
}
