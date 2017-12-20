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

	"fmt"
	"github.com/gorilla/websocket"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/yarpcerrors"
	"time"
)

type handler struct {
	i *Inbound
}

func newHandler(i *Inbound) *handler {
	return &handler{i: i}
}

func (h *handler) handle(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, nil, 400, 400)
	if err != nil {
		// TODO HANDLE
		return
	}

	go func() {
		defer conn.Close()
		transportRequest, err := h.getBasicTransportRequest(r)

		ctx := context.Background()
		handlerSpec, err := h.i.router.Choose(ctx, transportRequest)
		if err != nil {
			return
		}
		switch handlerSpec.Type() {
		case transport.Streaming:
			err = h.handleStream(ctx, transportRequest, handlerSpec.Stream(), conn)
			err = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(getControlMessage(err), "testtest"), time.Now().Add(time.Second))
			if err != nil {
				fmt.Println("something horrible went wrong: ", err)
			}
		}
		return
	}()
}

func getControlMessage(err error) int {
	if err == nil {
		return websocket.CloseNormalClosure
	}
	if yarpcerrors.FromError(err).Code() == yarpcerrors.CodeInternal {
		return websocket.CloseInternalServerErr
	}
	return websocket.CloseAbnormalClosure
}

func (h *handler) getBasicTransportRequest(r *http.Request) (*transport.Request, error) {
	transportRequest := &transport.Request{
		Caller:    r.Header["Rpc-Caller"][0],
		Service:   r.Header["Rpc-Service"][0],
		Procedure: r.Header["Rpc-Procedure"][0],
		Encoding:  transport.Encoding(r.Header["Rpc-Encoding"][0]),
	}
	return transportRequest, nil
}

func (h *handler) handleStream(
	ctx context.Context,
	transportRequest *transport.Request,
	streamHandler transport.StreamHandler,
	conn *websocket.Conn,
) error {
	sreq := &transport.StreamRequest{
		Meta: transportRequest.ToRequestMeta(),
	}
	stream, err := newServerStream(ctx, sreq, conn)
	if err != nil {
		return err
	}
	return transport.DispatchStreamHandler(
		streamHandler,
		stream,
	)
}

type serverStream struct {
	ctx  context.Context
	req  *transport.StreamRequest
	conn *websocket.Conn
}

func newServerStream(ctx context.Context, req *transport.StreamRequest, conn *websocket.Conn) (*transport.ServerStream, error) {
	return transport.NewServerStream(&serverStream{
		ctx:  ctx,
		req:  req,
		conn: conn,
	})
}

func (ss *serverStream) Context() context.Context {
	return ss.ctx
}

func (ss *serverStream) Request() *transport.StreamRequest {
	return ss.req
}

func (ss *serverStream) SendMessage(_ context.Context, m *transport.StreamMessage) error {
	msg, err := ioutil.ReadAll(m.Body)
	_ = m.Body.Close()
	if err != nil {
		return wrapError(err)
	}
	return wrapError(ss.conn.WriteMessage(websocket.BinaryMessage, msg))
}

func (ss *serverStream) ReceiveMessage(_ context.Context) (*transport.StreamMessage, error) {
	msgType, msg, err := ss.conn.ReadMessage()
	if err != nil {
		return nil, wrapError(err)
	}
	if msgType != websocket.BinaryMessage {
		return nil, yarpcerrors.InternalErrorf("invalid websocket message type %s", msgType)
	}
	return &transport.StreamMessage{Body: ioutil.NopCloser(bytes.NewReader(msg))}, nil
}
