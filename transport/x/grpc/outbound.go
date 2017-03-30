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
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/errors"
	internalsync "go.uber.org/yarpc/internal/sync"
)

var _ transport.UnaryOutbound = (*Outbound)(nil)

// Outbound is a transport.UnaryOutbound.
type Outbound struct {
	once       internalsync.LifecycleOnce
	lock       sync.Mutex
	address    string
	clientConn *grpc.ClientConn
}

// NewSingleOutbound returns a new Outbound for the given adrress.
func NewSingleOutbound(address string) *Outbound {
	return &Outbound{internalsync.Once(), sync.Mutex{}, address, nil}
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
	return []transport.Transport{}
}

// Call implements transport.UnaryOutbound#Call.
func (o *Outbound) Call(ctx context.Context, request *transport.Request) (*transport.Response, error) {
	start := time.Now()
	md, err := requestToMetadata(request)
	if err != nil {
		return nil, err
	}
	requestBody, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}
	fullMethod, err := prodecureNameToFullMethod(request.Procedure)
	if err != nil {
		return nil, err
	}
	var responseBody []byte
	responseMD := metadata.New(nil)
	if err := grpc.Invoke(
		metadata.NewContext(ctx, md),
		fullMethod,
		&requestBody,
		&responseBody,
		o.clientConn,
		grpc.Header(&responseMD),
	); err != nil {
		return nil, errorToGRPCError(ctx, request, start, err)
	}
	responseHeaders, err := getApplicationHeaders(responseMD)
	if err != nil {
		return nil, err
	}
	return &transport.Response{
		Body:    ioutil.NopCloser(bytes.NewBuffer(responseBody)),
		Headers: responseHeaders,
	}, nil
}

func (o *Outbound) start() error {
	// TODO: redial
	clientConn, err := grpc.Dial(
		o.address,
		grpc.WithInsecure(),
		// TODO: want to support default codec
		grpc.WithCodec(customCodec{}),
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

func requestToMetadata(request *transport.Request) (metadata.MD, error) {
	md := metadata.New(nil)
	if err := addCaller(md, request.Caller); err != nil {
		return nil, err
	}
	if err := addEncoding(md, request.Encoding); err != nil {
		return nil, err
	}
	if err := addApplicationHeaders(md, request.Headers); err != nil {
		return nil, err
	}
	return md, nil
}

func errorToGRPCError(ctx context.Context, request *transport.Request, start time.Time, err error) error {
	deadline, _ := ctx.Deadline()
	ttl := deadline.Sub(start)
	switch grpc.Code(err) {
	case codes.DeadlineExceeded:
		return errors.ClientTimeoutError(request.Service, request.Procedure, ttl)
	case codes.Unimplemented, codes.InvalidArgument, codes.NotFound:
		return errors.RemoteBadRequestError(grpc.ErrorDesc(err))
	case codes.Canceled, codes.AlreadyExists, codes.PermissionDenied,
		codes.Unauthenticated, codes.ResourceExhausted, codes.FailedPrecondition,
		codes.Aborted, codes.OutOfRange, codes.Internal,
		codes.Unavailable, codes.DataLoss, codes.Unknown:
		fallthrough
	default:
		return errors.RemoteUnexpectedError(grpc.ErrorDesc(err))
	}
}
