// Copyright (c) 2017 Uber Technologies, Inc.
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

package grpc

import (
	"bytes"
	"context"
	"io/ioutil"
	"strings"
	"sync"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	internalsync "go.uber.org/yarpc/internal/sync"
	"go.uber.org/yarpc/transport/x/grpc/grpcheader"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UserAgent is the User-Agent that will be set for requests.
// http://www.grpc.io/docs/guides/wire.html#user-agents
const UserAgent = "yarpc-go/" + yarpc.Version

var _ transport.UnaryOutbound = (*Outbound)(nil)

// Outbound is a transport.UnaryOutbound.
type Outbound struct {
	once            internalsync.LifecycleOnce
	lock            sync.Mutex
	t               *Transport
	address         string
	outboundOptions *outboundOptions
	clientConn      *grpc.ClientConn
}

func newSingleOutbound(t *Transport, address string, options ...OutboundOption) *Outbound {
	return &Outbound{internalsync.Once(), sync.Mutex{}, t, address, newOutboundOptions(options), nil}
}

// Start implements transport.Lifecycle#Start.
func (o *Outbound) Start() error {
	return o.once.Start(o.start)
}

// Stop implements transport.Lifecycle#Stop.
func (o *Outbound) Stop() error {
	return o.once.Stop(o.stop)
}

// IsRunning implements transport.Lifecycle#IsRunning.
func (o *Outbound) IsRunning() bool {
	return o.once.IsRunning()
}

// Transports implements transport.Inbound#Transports.
func (o *Outbound) Transports() []transport.Transport {
	return []transport.Transport{o.t}
}

// Call implements transport.UnaryOutbound#Call.
func (o *Outbound) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	if err := o.once.WhenRunning(ctx); err != nil {
		return nil, err
	}
	var responseBody []byte
	responseMD := metadata.New(nil)
	invokeErr := o.invoke(ctx, request, &responseBody, &responseMD)
	responseHeaders, err := getApplicationHeaders(responseMD)
	if err != nil {
		return nil, err
	}
	return &transport.Response{
		Body:    ioutil.NopCloser(bytes.NewBuffer(responseBody)),
		Headers: responseHeaders,
	}, invokeErrorToYARPCError(invokeErr, responseMD)
}

func (o *Outbound) invoke(
	ctx context.Context,
	request *transport.Request,
	responseBody *[]byte,
	responseMD *metadata.MD,
) error {
	md, err := transportRequestToMetadata(request)
	if err != nil {
		return err
	}
	// TODO: use pooled buffers
	requestBody, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return err
	}
	fullMethod, err := procedureNameToFullMethod(request.Procedure)
	if err != nil {
		return err
	}
	var callOptions []grpc.CallOption
	if responseMD != nil {
		callOptions = []grpc.CallOption{grpc.Trailer(responseMD)}
	}
	return grpc.Invoke(
		metadata.NewContext(ctx, md),
		fullMethod,
		requestBody,
		responseBody,
		o.clientConn,
		callOptions...,
	)
}

func (o *Outbound) start() error {
	// TODO: redial
	clientConn, err := grpc.Dial(
		o.address,
		grpc.WithInsecure(),
		grpc.WithCodec(customCodec{}),
		// TODO: does this actually work for yarpc
		// this needs a lot of review
		//grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(o.outboundOptions.getTracer())),
		grpc.WithUserAgent(UserAgent),
	)
	if err != nil {
		return err
	}
	o.lock.Lock()
	defer o.lock.Unlock()
	o.clientConn = clientConn
	return nil
}

func (o *Outbound) stop() error {
	o.lock.Lock()
	defer o.lock.Unlock()
	if o.clientConn != nil {
		return o.clientConn.Close()
	}
	return nil
}

func invokeErrorToYARPCError(err error, responseMD metadata.MD) error {
	if err == nil {
		return nil
	}
	if yarpc.IsYARPCError(err) {
		return err
	}
	status, ok := status.FromError(err)
	// if not a yarpc error or grpc error, just return the error
	if !ok {
		return err
	}
	code, ok := GRPCCodeToCode[status.Code()]
	if !ok {
		code = yarpc.CodeUnknown
	}
	var name string
	if responseMD != nil {
		value, ok := responseMD[grpcheader.ErrorNameHeader]
		// TODO: what to do if the length is > 1?
		if ok && len(value) == 1 {
			name = value[0]
		}
	}
	message := status.Message()
	// we put the name as a prefix for grpc compatibility
	// if there was no message, the message will be the name, so we leave it as the message
	if name != "" && message != "" && message != name {
		message = strings.TrimPrefix(message, name+": ")
	} else if name != "" && message == name {
		message = ""
	}
	return transport.FromHeaders(transport.Code(code), name, message)
}
