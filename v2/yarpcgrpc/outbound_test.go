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

package yarpcgrpc

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcpeer"
	"go.uber.org/yarpc/v2/yarpctest"
	"google.golang.org/grpc"
)

func TestNoRequest(t *testing.T) {
	out := &Outbound{}
	_, _, err := out.Call(context.Background(), nil, &yarpc.Buffer{})
	assert.Equal(t, yarpcerror.InvalidArgumentErrorf("request for grpc outbound was nil"), err)
}

func TestCallStreamWithNoRequest(t *testing.T) {
	out := &Outbound{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	_, err := out.CallStream(ctx, nil)

	require.Contains(t, err.Error(), yarpcerror.InvalidArgumentErrorf("stream request requires a yarpc.Request").Error())
}

func TestCallStreamWithInvalidHeader(t *testing.T) {
	out := &Outbound{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	req := &yarpc.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  yarpc.Encoding("raw"),
		Procedure: "proc",
		Headers:   yarpc.NewHeaders().With("rpc-caller", "reserved header"),
	}
	_, err := out.CallStream(ctx, req)

	require.Contains(t, err.Error(), yarpcerror.InvalidArgumentErrorf("cannot use reserved header in application headers: rpc-caller").Error())
}

func TestCallStreamWithInvalidProcedure(t *testing.T) {
	out := &Outbound{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	req := &yarpc.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  yarpc.Encoding("raw"),
		Procedure: "",
	}
	_, err := out.CallStream(ctx, req)

	require.Contains(t, err.Error(), yarpcerror.InvalidArgumentErrorf("invalid procedure name: ").Error())
}

func TestCallStreamWithChooserError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	chooser := yarpctest.NewMockChooser(mockCtrl)
	chooser.EXPECT().Choose(gomock.Any(), gomock.Any()).Return(nil, nil, yarpcerror.InternalErrorf("error"))

	out := &Outbound{Chooser: chooser}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	req := &yarpc.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  yarpc.Encoding("raw"),
		Procedure: "proc",
	}
	_, err := out.CallStream(ctx, req)

	require.Contains(t, err.Error(), yarpcerror.InternalErrorf("error").Error())
}

func TestCallStreamWithInvalidPeer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	fakePeer := yarpctest.NewMockPeer(mockCtrl)
	chooser := yarpctest.NewMockChooser(mockCtrl)
	chooser.EXPECT().Choose(gomock.Any(), gomock.Any()).Return(fakePeer, func(error) {}, nil)

	out := &Outbound{Chooser: chooser}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	req := &yarpc.Request{
		Caller:    "caller",
		Service:   "service",
		Encoding:  yarpc.Encoding("raw"),
		Procedure: "proc",
	}
	_, err := out.CallStream(ctx, req)

	require.Contains(
		t,
		err.Error(),
		yarpcpeer.ErrInvalidPeerConversion{
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
					mdWriter := newMetadataWriter()

					if tt.headerKey != "" {
						mdWriter.AddSystemHeader(tt.headerKey, tt.headerValue)
					}
					// Send the response attributes back and end the stream.
					if sendErr := stream.SendMsg(&empty.Empty{}); sendErr != nil {
						// We couldn't send the response.
						return sendErr
					}

					stream.SetTrailer(mdWriter.MD())
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

			u, err := url.Parse("http://" + listener.Addr().String())
			require.NoError(t, err)

			dialer := &Dialer{}
			out := &Outbound{
				Dialer: dialer,
				URL:    u,
			}
			dialer.Start(context.Background())
			defer dialer.Stop(context.Background())

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			_, _, err = out.Call(ctx, &yarpc.Request{
				Service:   "Service",
				Procedure: "Hello",
			}, yarpc.NewBufferString("world"))

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "does not match")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
