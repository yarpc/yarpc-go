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

func TestCallServiceMatch(t *testing.T) {
	tests := []struct {
		msg         string
		headerKey   string
		headerValue string
		wantErr     bool
	}{
		{
			msg:         "call service match success",
			headerKey:   ServiceHeader,
			headerValue: "Service",
		},
		{
			msg:         "call service match failed",
			headerKey:   ServiceHeader,
			headerValue: "ThisIsWrongSvcName",
			wantErr:     true,
		},
		{
			msg: "no service name response header",
		},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			server := grpc.NewServer(
				grpc.UnknownServiceHandler(func(srv interface{}, stream grpc.ServerStream) error {
					responseWriter := newResponseWriter(&transport.Request{})

					if tt.headerKey != "" {
						responseWriter.AddSystemHeader(tt.headerKey, tt.headerValue)
					}

					// Send the response attributes back and end the stream.
					if sendErr := stream.SendMsg(&empty.Empty{}); sendErr != nil {
						// We couldn't send the response.
						return sendErr
					}
					responseWriter.Close(stream)
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
				Service:   "Service",
				Procedure: "Hello",
				Body:      bytes.NewReader([]byte("world")),
			}
			_, err = out.Call(ctx, req)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "does not match")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
