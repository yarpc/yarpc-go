// Copyright (c) 2018 Uber Technologies, Inc.
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
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/api/peer/peertest"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestNoRequest(t *testing.T) {
	tran := NewTransport()
	out := tran.NewSingleOutbound("localhost:0")

	_, err := out.Call(context.Background(), nil)
	assert.Equal(t, yarpcerrors.InvalidArgumentErrorf("request for grpc outbound was nil"), err)
}

func TestCallStreamWhenNotRunning(t *testing.T) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:0"))
	require.NoError(t, err)

	tran := NewTransport()
	out := tran.NewSingleOutbound(listener.Addr().String())

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()
	_, err = out.CallStream(ctx, &transport.StreamRequest{})

	require.Contains(t, err.Error(), context.DeadlineExceeded.Error())
}

func TestCallStreamWithNoRequestMeta(t *testing.T) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:0"))
	require.NoError(t, err)

	tran := NewTransport()
	out := tran.NewSingleOutbound(listener.Addr().String())
	require.NoError(t, tran.Start())
	require.NoError(t, out.Start())
	defer tran.Stop()
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	_, err = out.CallStream(ctx, &transport.StreamRequest{})

	require.Contains(t, err.Error(), yarpcerrors.InvalidArgumentErrorf("stream request requires a request metadata").Error())
}

func TestCallStreamWithInvalidHeader(t *testing.T) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:0"))
	require.NoError(t, err)

	tran := NewTransport()
	out := tran.NewSingleOutbound(listener.Addr().String())
	require.NoError(t, tran.Start())
	require.NoError(t, out.Start())
	defer tran.Stop()
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	req := &transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Caller:    "caller",
			Service:   "service",
			Encoding:  transport.Encoding("raw"),
			Procedure: "proc",
			Headers:   transport.NewHeaders().With("rpc-caller", "reserved header"),
		},
	}
	_, err = out.CallStream(ctx, req)

	require.Contains(t, err.Error(), yarpcerrors.InvalidArgumentErrorf("cannot use reserved header in application headers: rpc-caller").Error())
}

func TestCallStreamWithInvalidProcedure(t *testing.T) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:0"))
	require.NoError(t, err)

	tran := NewTransport()
	out := tran.NewSingleOutbound(listener.Addr().String())
	require.NoError(t, tran.Start())
	require.NoError(t, out.Start())
	defer tran.Stop()
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	req := &transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Caller:    "caller",
			Service:   "service",
			Encoding:  transport.Encoding("raw"),
			Procedure: "",
		},
	}
	_, err = out.CallStream(ctx, req)

	require.Contains(t, err.Error(), yarpcerrors.InvalidArgumentErrorf("invalid procedure name: ").Error())
}

func TestCallStreamWithChooserError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	chooser := peertest.NewMockChooser(mockCtrl)
	chooser.EXPECT().Start()
	chooser.EXPECT().Stop()
	chooser.EXPECT().Choose(gomock.Any(), gomock.Any()).Return(nil, nil, yarpcerrors.InternalErrorf("error"))

	tran := NewTransport()
	out := tran.NewOutbound(chooser)

	require.NoError(t, tran.Start())
	require.NoError(t, out.Start())
	defer tran.Stop()
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	req := &transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Caller:    "caller",
			Service:   "service",
			Encoding:  transport.Encoding("raw"),
			Procedure: "proc",
		},
	}
	_, err := out.CallStream(ctx, req)

	require.Contains(t, err.Error(), yarpcerrors.InternalErrorf("error").Error())
}

func TestCallStreamWithInvalidPeer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	fakePeer := peertest.NewMockPeer(mockCtrl)
	chooser := peertest.NewMockChooser(mockCtrl)
	chooser.EXPECT().Start()
	chooser.EXPECT().Stop()
	chooser.EXPECT().Choose(gomock.Any(), gomock.Any()).Return(fakePeer, func(error) {}, nil)

	tran := NewTransport()
	out := tran.NewOutbound(chooser)

	require.NoError(t, tran.Start())
	require.NoError(t, out.Start())
	defer tran.Stop()
	defer out.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	req := &transport.StreamRequest{
		Meta: &transport.RequestMeta{
			Caller:    "caller",
			Service:   "service",
			Encoding:  transport.Encoding("raw"),
			Procedure: "proc",
		},
	}
	_, err := out.CallStream(ctx, req)

	require.Contains(
		t,
		err.Error(),
		peer.ErrInvalidPeerConversion{
			Peer:         fakePeer,
			ExpectedType: "*grpcPeer",
		}.Error(),
	)
}

func TestCallServiceMatchAndUUIDMatch(t *testing.T) {
	const (
		rightServiceName = "yarpc-service"
		wrongServiceName = "wrong-service"
		wrongUuid        = "dead-cafe-beef-1234"
	)
	tests := []struct {
		msg          string
		headers      map[string]string
		sendSameUUID bool
		wantErr      string
	}{
		{
			msg: "service match success and uuid match success",
			headers: map[string]string{
				ServiceHeader: rightServiceName,
			},
			sendSameUUID: true,
		},
		{
			msg: "service match success and uuid match failure",
			headers: map[string]string{
				ServiceHeader:     rightServiceName,
				RequestUUIDHeader: wrongUuid,
			},
			wantErr: "does not match the uuid",
		},
		{
			msg: "service match failure and uuid match success",
			headers: map[string]string{
				ServiceHeader: wrongServiceName,
			},
			sendSameUUID: true,
			wantErr:      "does not match the service name",
		},
		{
			msg: "service match failure and uuid match failure",
			headers: map[string]string{
				ServiceHeader:     wrongServiceName,
				RequestUUIDHeader: wrongUuid,
			},
			wantErr: "does not match the service name",
		},
		{
			msg: "service name missing and uuid missing",
		},
		{
			msg: "service name missing and uuid match failure",
			headers: map[string]string{
				RequestUUIDHeader: wrongUuid,
			},
			wantErr: "does not match the uuid",
		},
		{
			msg:          "service name missing and uuid match success",
			sendSameUUID: true,
		},
		{
			msg: "service name success and uuid missing",
			headers: map[string]string{
				ServiceHeader: rightServiceName,
			},
		},
		{
			msg: "service name match failure and uuid missing",
			headers: map[string]string{
				ServiceHeader: wrongServiceName,
			},
			wantErr: "does not match the service name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			server := grpc.NewServer(
				grpc.UnknownServiceHandler(func(srv interface{}, stream grpc.ServerStream) error {
					responseWriter := newResponseWriter()
					defer responseWriter.Close()
					ctx := stream.Context()
					md, ok := metadata.FromIncomingContext(ctx)
					if md == nil || !ok {
						return yarpcerrors.Newf(yarpcerrors.CodeInternal, "cannot get metadata from ctx: %v", ctx)
					}

					if tt.sendSameUUID {
						if values, ok := md[RequestUUIDHeader]; ok {
							responseWriter.AddSystemHeader(RequestUUIDHeader, values[0])
						} else {
							return yarpcerrors.Newf(yarpcerrors.CodeInternal, "cannot get uuid header value")
						}
					}

					for headerKey, headerVal := range tt.headers {
						responseWriter.AddSystemHeader(headerKey, headerVal)
					}

					// Send the response attributes back and end the stream.
					if sendErr := stream.SendMsg(&empty.Empty{}); sendErr != nil {
						// We couldn't send the response.
						return sendErr
					}
					if responseWriter.md != nil {
						stream.SetTrailer(responseWriter.md)
					}
					return nil
				}),
			)
			listener, err := net.Listen("tcp", ":0")
			require.NoError(t, err)
			go func() {
				err := server.Serve(listener)
				require.NoError(t, err)
			}()
			defer server.Stop()

			grpcTransport := NewTransport()
			out := grpcTransport.NewSingleOutbound(listener.Addr().String())
			require.NoError(t, grpcTransport.Start())
			require.NoError(t, out.Start())
			defer grpcTransport.Stop()
			defer out.Stop()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			req := &transport.Request{
				Service:   rightServiceName,
				Procedure: "Hello",
				Body:      bytes.NewReader([]byte("world")),
			}
			_, err = out.Call(ctx, req)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
