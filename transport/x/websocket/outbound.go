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

package websocket

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"sync"
	//"time"

	"github.com/gorilla/websocket"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/pkg/lifecycle"
	"go.uber.org/yarpc/yarpcerrors"
	"io"
)

var _ transport.StreamOutbound = (*Outbound)(nil)

// Outbound is a transport.UnaryOutbound.
type Outbound struct {
	once    *lifecycle.Once
	lock    sync.Mutex
	t       *Transport
	address string
}

func newSingleOutbound(t *Transport, address string) *Outbound {
	return &Outbound{
		once:    lifecycle.NewOnce(),
		t:       t,
		address: address,
	}
}

// Start implements transport.Lifecycle#Start.
func (o *Outbound) Start() error {
	return nil
}

// Stop implements transport.Lifecycle#Stop.
func (o *Outbound) Stop() error {
	return nil
}

// IsRunning implements transport.Lifecycle#IsRunning.
func (o *Outbound) IsRunning() bool {
	return o.once.IsRunning()
}

// Transports implements transport.Inbound#Transports.
func (o *Outbound) Transports() []transport.Transport {
	return []transport.Transport{o.t}
}

// CallStream implements transport.StreamOutbound#CallStream.
func (o *Outbound) CallStream(ctx context.Context, request *transport.StreamRequest) (*transport.ClientStream, error) {
	return o.stream(ctx, request)
}

func (o *Outbound) stream(
	ctx context.Context,
	request *transport.StreamRequest,
) (_ *transport.ClientStream, retErr error) {
	header := http.Header{}
	header.Add("rpc-procedure", request.Meta.Procedure)
	header.Add("rpc-service", request.Meta.Service)
	header.Add("rpc-encoding", string(request.Meta.Encoding))
	header.Add("rpc-caller", string(request.Meta.Caller))
	conn, _, err := websocket.DefaultDialer.Dial(o.address, header)
	if err != nil {
		return nil, err
	}

	return newClientStream(ctx, request, conn)
}

type clientStream struct {
	ctx  context.Context
	treq *transport.StreamRequest
	conn *websocket.Conn
}

func newClientStream(ctx context.Context, treq *transport.StreamRequest, conn *websocket.Conn) (*transport.ClientStream, error) {
	return transport.NewClientStream(&clientStream{
		ctx:  ctx,
		treq: treq,
		conn: conn,
	})
}

func (ss *clientStream) Context() context.Context {
	return ss.ctx
}

func (ss *clientStream) Request() *transport.StreamRequest {
	return ss.treq
}

func (ss *clientStream) SendMessage(ctx context.Context, m *transport.StreamMessage) error {
	msg, err := ioutil.ReadAll(m.Body)
	if err != nil {
		return wrapError(err)
	}
	return wrapError(ss.conn.WriteMessage(websocket.BinaryMessage, msg))
}

func (ss *clientStream) ReceiveMessage(context.Context) (*transport.StreamMessage, error) {
	msgType, msg, err := ss.conn.ReadMessage()
	if err != nil {
		return nil, wrapError(err)
	}
	if msgType != websocket.BinaryMessage {
		return nil, yarpcerrors.InternalErrorf("invalid websocket message type %s", msgType)
	}
	return &transport.StreamMessage{Body: ioutil.NopCloser(bytes.NewReader(msg))}, nil
}

func (cs *clientStream) Close(context.Context) error {
	return cs.conn.Close()
	//cs.conn.WriteControl()
	//return cs.conn.WriteControl(websocket.CloseMessage, nil, time.Now().Add(time.Second))
}

func wrapError(err error) error {
	if closeErr, ok := err.(*websocket.CloseError); ok {
		if closeErr.Code == websocket.CloseNormalClosure || closeErr.Code == websocket.CloseNoStatusReceived {
			return io.EOF
		}
		if closeErr.Code == websocket.CloseInternalServerErr {
			return yarpcerrors.InternalErrorf(closeErr.Error())
		}
		return yarpcerrors.UnknownErrorf(closeErr.Error())
	}
	return err
}
