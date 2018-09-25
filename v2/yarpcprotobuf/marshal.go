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

package yarpcprotobuf

import (
	"bytes"
	"io"
	"sync"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"go.uber.org/yarpc/internal/bufferpool"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcjson"
)

var (
	_jsonMarshaler   = &jsonpb.Marshaler{}
	_jsonUnmarshaler = &jsonpb.Unmarshaler{AllowUnknownFields: true}
	_bufferPool      = sync.Pool{
		New: func() interface{} {
			return proto.NewBuffer(make([]byte, 1024))
		},
	}
)

func unmarshal(encoding yarpc.Encoding, reader io.Reader, message proto.Message) error {
	buf := bufferpool.Get()
	defer bufferpool.Put(buf)
	if _, err := buf.ReadFrom(reader); err != nil {
		return err
	}
	body := buf.Bytes()
	if len(body) == 0 {
		return nil
	}
	switch encoding {
	case Encoding:
		return proto.Unmarshal(body, message)
	case yarpcjson.Encoding:
		return _jsonUnmarshaler.Unmarshal(bytes.NewReader(body), message)
	default:
		return yarpcerror.Newf(yarpcerror.CodeInternal, "encoding.Expect should have handled encoding %q but did not", encoding)
	}
}

func marshal(encoding yarpc.Encoding, message proto.Message) ([]byte, func(), error) {
	switch encoding {
	case Encoding:
		return marshalProto(message)
	case yarpcjson.Encoding:
		return marshalJSON(message)
	default:
		return nil, nil, yarpcerror.Newf(yarpcerror.CodeInternal, "encoding.Expect should have handled encoding %q but did not", encoding)
	}
}

func marshalProto(message proto.Message) ([]byte, func(), error) {
	protoBuffer := getBuffer()
	cleanup := func() { putBuffer(protoBuffer) }
	if err := protoBuffer.Marshal(message); err != nil {
		cleanup()
		return nil, nil, err
	}
	return protoBuffer.Bytes(), cleanup, nil
}

func marshalJSON(message proto.Message) ([]byte, func(), error) {
	buf := bufferpool.Get()
	cleanup := func() { bufferpool.Put(buf) }
	if err := _jsonMarshaler.Marshal(buf, message); err != nil {
		cleanup()
		return nil, nil, err
	}
	return buf.Bytes(), cleanup, nil
}

func getBuffer() *proto.Buffer {
	buf := _bufferPool.Get().(*proto.Buffer)
	buf.Reset()
	return buf
}

func putBuffer(buf *proto.Buffer) {
	_bufferPool.Put(buf)
}
