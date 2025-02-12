// Copyright (c) 2025 Uber Technologies, Inc.
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

package v2_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/encoding/protobuf/internal/testpb/v2"
	v2 "go.uber.org/yarpc/encoding/protobuf/v2"
	"go.uber.org/yarpc/internal/clientconfig"
	"go.uber.org/yarpc/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/yarpctest"
	"google.golang.org/protobuf/proto"
)

var protocolOptionsTable = []struct {
	msg  string
	opts []v2.ClientOption
}{
	{
		msg: "protobuf",
	},
	{
		msg: "json",
		opts: []v2.ClientOption{
			v2.UseJSON,
		},
	},
}

func TestUnary(t *testing.T) {
	for _, tt := range protocolOptionsTable {
		t.Run(tt.msg, func(t *testing.T) {
			server := &testServer{}
			procedures := testpb.BuildTestYARPCProcedures(server)
			router := yarpc.NewMapRouter("test")
			router.Register(procedures)

			trans := yarpctest.NewFakeTransport()
			pc := peer.NewSingle(hostport.Identify("1"), trans)
			ob := trans.NewOutbound(pc, yarpctest.OutboundRouter(router))
			cc := clientconfig.MultiOutbound("test", "test", transport.Outbounds{
				Unary: ob,
			})
			client := testpb.NewTestYARPCClient(cc, tt.opts...)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sent := &testpb.TestMessage{Value: "echo"}
			received, err := client.Unary(ctx, sent)
			require.NoError(t, err)
			assert.True(t, proto.Equal(sent, received))
		})
	}
}

var _ testpb.TestYARPCServer = (*testServer)(nil)

// testServer provides unary and streaming echo method implementations.
type testServer struct{}

func (s *testServer) Unary(ctx context.Context, msg *testpb.TestMessage) (*testpb.TestMessage, error) {
	return msg, nil
}

func (s *testServer) Duplex(str testpb.TestServiceDuplexYARPCServer) error {
	for {
		msg, err := str.Recv()
		if err != nil {
			return err
		}
		if msg.Value == "please explode" {
			return errors.New("explosion occurred")
		}
		err = str.Send(msg)
		if err != nil {
			return err
		}
	}
}

func TestDuplexStream(t *testing.T) {
	for _, tt := range protocolOptionsTable {
		t.Run(tt.msg, func(t *testing.T) {
			server := &testServer{}
			procedures := testpb.BuildTestYARPCProcedures(server)
			router := yarpc.NewMapRouter("test")
			router.Register(procedures)

			trans := yarpctest.NewFakeTransport()
			pc := peer.NewSingle(hostport.Identify("1"), trans)
			ob := trans.NewOutbound(pc, yarpctest.OutboundRouter(router))
			cc := clientconfig.MultiOutbound("test", "test", transport.Outbounds{
				Stream: ob,
			})
			client := testpb.NewTestYARPCClient(cc, tt.opts...)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			str, err := client.Duplex(ctx)
			require.NoError(t, err)

			// Send a message.
			sent := &testpb.TestMessage{Value: "echo"}
			{
				err := str.Send(sent)
				require.NoError(t, err)
			}

			// Receive the echoed message.
			{
				msg, err := str.Recv()
				require.NoError(t, err)
				assert.True(t, proto.Equal(sent, msg))
			}

			// Close the client side of the stream.
			str.CloseSend()

			// Verify that the server closes as well.
			{
				_, err := str.Recv()
				require.Equal(t, err, io.EOF)
			}
		})
	}
}

func TestStreamServerError(t *testing.T) {
	table := []struct {
		msg          string
		clientCloses bool
	}{
		{
			msg:          "client closes after error",
			clientCloses: true,
		},
		{
			msg:          "client returns immediately after error",
			clientCloses: false,
		},
	}

	for _, ct := range table {
		t.Run(ct.msg, func(t *testing.T) {
			for _, tt := range protocolOptionsTable {
				t.Run(tt.msg, func(t *testing.T) {
					server := &testServer{}
					procedures := testpb.BuildTestYARPCProcedures(server)
					router := yarpc.NewMapRouter("test")
					router.Register(procedures)

					trans := yarpctest.NewFakeTransport()
					pc := peer.NewSingle(hostport.Identify("1"), trans)
					ob := trans.NewOutbound(pc, yarpctest.OutboundRouter(router))
					cc := clientconfig.MultiOutbound("test", "test", transport.Outbounds{
						Stream: ob,
					})
					client := testpb.NewTestYARPCClient(cc, tt.opts...)

					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()

					str, err := client.Duplex(ctx)
					require.NoError(t, err)

					// Send a message.
					sent := &testpb.TestMessage{Value: "please explode"}
					{
						err := str.Send(sent)
						require.NoError(t, err)
					}

					// Receive the handler error on next receive.
					{
						msg, err := str.Recv()
						require.Error(t, err)
						assert.Nil(t, msg)
					}

					// Close the client side of the stream.
					if ct.clientCloses {
						str.CloseSend()
					}

					// Verify that the server closes as well.
					{
						msg, err := str.Recv()
						require.Equal(t, err, io.EOF)
						assert.Nil(t, msg)
					}
				})
			}
		})
	}
}
