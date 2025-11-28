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

package protobuf

import (
	"bytes"
	"context"

	"github.com/gogo/protobuf/proto"
	"go.uber.org/yarpc/api/transport"
	"google.golang.org/grpc/mem"
)

type poolWrapper struct {
	cleanup func()
}

func (p *poolWrapper) Get(length int) *[]byte { return nil }

func (p *poolWrapper) Put(buf *[]byte) {
	if p.cleanup != nil {
		p.cleanup()
	}
}

// readFromStream reads a proto.Message from a stream.
func readFromStream(
	ctx context.Context,
	stream transport.Stream,
	newMessage func() proto.Message,
	codec *codec,
) (proto.Message, error) {
	streamMsg, err := stream.ReceiveMessage(ctx)
	if err != nil {
		return nil, convertFromYARPCError(Encoding, err, codec)
	}
	message := newMessage()
	if err := unmarshal(stream.Request().Meta.Encoding, streamMsg.Body, message, codec); err != nil {
		streamMsg.Body.Close()
		return nil, err
	}
	if streamMsg.Body != nil {
		streamMsg.Body.Close()
	}
	return message, nil
}

// writeToStream writes a proto.Message to a stream.
func writeToStream(ctx context.Context, stream transport.Stream, message proto.Message, codec *codec) error {
	messageData, cleanup, err := marshal(stream.Request().Meta.Encoding, message, codec)
	if err != nil {
		return err
	}

	buffer := mem.NewBuffer(&messageData, &poolWrapper{cleanup: cleanup})

	return stream.SendMessage(
		ctx,
		&transport.StreamMessage{
			Body: &bufferReadCloser{
				buffer: buffer,
				Reader: bytes.NewReader(messageData),
			},
			BodySize: len(messageData),
		},
	)
}

type bufferReadCloser struct {
	buffer mem.Buffer
	*bytes.Reader
}

func (b *bufferReadCloser) Buffer() mem.Buffer {
	return b.buffer
}

func (b *bufferReadCloser) Close() error {
	b.buffer.Free()
	return nil
}
