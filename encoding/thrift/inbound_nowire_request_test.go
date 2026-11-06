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

package thrift

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/thriftrw/protocol/binary"
	"go.uber.org/thriftrw/protocol/stream"
	"go.uber.org/thriftrw/wire"
)

// requestDecodeHandler decodes the request through NoWireCall, as generated
// handlers do.
type requestDecodeHandler struct {
	body   stream.BodyReader
	reader io.Reader
}

func (h *requestDecodeHandler) HandleNoWire(ctx context.Context, nwc *NoWireCall) (NoWireResponse, error) {
	h.reader = nwc.Reader
	_, err := nwc.RequestReader.ReadRequest(ctx, nwc.EnvelopeType, nwc.Reader, h.body)
	return NoWireResponse{}, err
}

// bodyOnlyRequestReader decodes a bare body from the binary protocol, with no
// envelope, using the reader it is given.
type bodyOnlyRequestReader struct{}

func (bodyOnlyRequestReader) ReadRequest(_ context.Context, _ wire.EnvelopeType, r io.Reader, body stream.BodyReader) (stream.ResponseWriter, error) {
	sr := binary.Default.Reader(r)
	defer sr.Close()
	return nil, body.Decode(sr)
}

func TestNoWireHandlerRequestReads(t *testing.T) {
	values := make([]string, 1280)
	for i := range values {
		values[i] = strings.Repeat("x", 96)
	}
	payload := encodeStringList(t, values)
	readError := errors.New("request read failed")
	tests := []struct {
		name       string
		reader     io.Reader
		wantError  error
		batchReads bool
	}{
		{name: "stream", reader: bytes.NewReader(payload), batchReads: true},
		{name: "short reads", reader: iotest.OneByteReader(bytes.NewReader(payload))},
		{name: "data with EOF", reader: iotest.DataErrReader(bytes.NewReader(payload))},
		{name: "truncated", reader: bytes.NewReader(payload[:len(payload)-1]), wantError: io.ErrUnexpectedEOF},
		{name: "read failure", reader: io.MultiReader(bytes.NewReader(payload[:128]), iotest.ErrReader(readError)), wantError: readError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result stringListResponse
			body := &trackedResponseBody{Reader: tt.reader}
			treq := request()
			treq.Body = body
			h := thriftNoWireHandler{
				Handler:       &requestDecodeHandler{body: &result},
				RequestReader: bodyOnlyRequestReader{},
			}
			err := h.HandleOneway(context.Background(), treq)
			assert.Equal(t, 1, body.closes, "the transport body must close on success and decode failure")
			if tt.wantError != nil {
				require.ErrorIs(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, values, []string(result))
			if tt.batchReads {
				// An unbuffered decoder makes at least two reads per string.
				// Leave room for refills without coupling to an exact buffer size.
				assert.Less(t, body.reads, len(values)/10)
			}
		})
	}
}

// ThriftRW's binary Protocol.ReadRequest sniffs for an envelope with a single
// two-byte Read. A transport body may return a single byte first, which must
// not be mistaken for an empty-struct request.
func TestNoWireHandlerEnvelopedRequestWithShortFirstRead(t *testing.T) {
	values := []string{"first", "second", "third"}
	var payload bytes.Buffer
	sw := binary.Default.Writer(&payload)
	require.NoError(t, sw.WriteEnvelopeBegin(stream.EnvelopeHeader{Name: "someMethod", Type: wire.OneWay, SeqID: 1}))
	require.NoError(t, stringListRequest(values).Encode(sw))
	require.NoError(t, sw.WriteEnvelopeEnd())
	require.NoError(t, sw.Close())

	var result stringListResponse
	treq := request()
	treq.Body = &trackedResponseBody{Reader: iotest.OneByteReader(bytes.NewReader(payload.Bytes()))}
	h := thriftNoWireHandler{
		Handler:       &requestDecodeHandler{body: &result},
		RequestReader: binary.Default,
	}
	require.NoError(t, h.HandleOneway(context.Background(), treq))
	assert.Equal(t, values, []string(result))
}

func TestNoWireHandlerRequestAfterDecodeFailure(t *testing.T) {
	decodeError := errors.New("decode rejected request")
	h := thriftNoWireHandler{
		RequestReader: bodyOnlyRequestReader{},
	}
	for _, value := range []string{"first", "second", "third"} {
		treq := request()
		treq.Body = &trackedResponseBody{Reader: bytes.NewReader(encodeStringList(t, []string{value}))}
		h.Handler = &requestDecodeHandler{body: responseDecoderFunc(func(sr stream.Reader) error {
			// Leave unread data behind in the pooled reader.
			if _, err := sr.ReadListBegin(); err != nil {
				return err
			}
			return decodeError
		})}
		require.ErrorIs(t, h.HandleOneway(context.Background(), treq), decodeError)

		// A reused pooled reader must not leak data from the failed request.
		var decoded stringListResponse
		treq = request()
		treq.Body = &trackedResponseBody{Reader: bytes.NewReader(encodeStringList(t, []string{value}))}
		h.Handler = &requestDecodeHandler{body: &decoded}
		require.NoError(t, h.HandleOneway(context.Background(), treq))
		assert.Equal(t, []string{value}, []string(decoded))
	}
}

func TestNoWireHandlerSeekableRequest(t *testing.T) {
	body := &seekableResponseBody{Reader: bytes.NewReader(encodeStringList(t, []string{"skipped"}))}
	treq := request()
	treq.Body = body
	handler := &requestDecodeHandler{body: responseDecoderFunc(func(sr stream.Reader) error { return sr.Skip(wire.TList) })}
	h := thriftNoWireHandler{
		Handler:       handler,
		RequestReader: bodyOnlyRequestReader{},
	}
	require.NoError(t, h.HandleOneway(context.Background(), treq))
	assert.Positive(t, body.seeks, "seekable requests must retain the protocol's zero-copy skip path")
	assert.Same(t, body, handler.reader, "seekable requests must not be wrapped")
	assert.Equal(t, 1, body.closes)
}
