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

package yarpc

import (
	js "encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yarpc/yarpc-go"
	"github.com/yarpc/yarpc-go/encoding/json"
	"github.com/yarpc/yarpc-go/transport"
	ht "github.com/yarpc/yarpc-go/transport/http"
	tch "github.com/yarpc/yarpc-go/transport/tchannel"

	"github.com/uber/tchannel-go"
	"golang.org/x/net/context"
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
func Phone(reqMeta yarpc.ReqMeta, body *PhoneRequest) (*PhoneResponse, yarpc.ResMeta, error) {
	var outbound transport.Outbound

	switch {
	case body.Transport.HTTP != nil:
		cl := &http.Client{Transport: new(http.Transport)}
		// ^See crossdock/client/rpc/rpc.go for explanation

		t := body.Transport.HTTP
		url := fmt.Sprintf("http://%s:%d", t.Host, t.Port)
		outbound = ht.NewOutboundWithClient(url, cl)
	case body.Transport.TChannel != nil:
		t := body.Transport.TChannel
		hostport := fmt.Sprintf("%s:%d", t.Host, t.Port)
		ch, err := tchannel.NewChannel("yarpc-test-client", nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build TChannel: %v", err)
		}
		outbound = tch.NewOutbound(ch, tch.HostPort(hostport))
	default:
		return nil, nil, fmt.Errorf("unconfigured transport")
	}

	client := json.New(transport.Channel{
		Caller:   "yarpc-test", // TODO use reqMeta.Service,
		Service:  body.Service,
		Outbound: outbound,
	})

	resBody := PhoneResponse{
		Service:   "yarpc-test", // TODO use reqMeta.Service
		Procedure: reqMeta.Procedure(),
	}

	ctx, _ := context.WithTimeout(reqMeta.Context(), 500*time.Millisecond)
	_, err := client.Call(
		yarpc.NewReqMeta(ctx).Procedure(body.Procedure),
		body.Body,
		&resBody.Body)
	if err != nil {
		return nil, nil, err
	}

	return &resBody, nil, nil
}
