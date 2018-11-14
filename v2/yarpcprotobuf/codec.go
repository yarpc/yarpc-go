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
	"sync"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
)

var (
	_jsonMarshaler   = &jsonpb.Marshaler{}
	_jsonUnmarshaler = &jsonpb.Unmarshaler{AllowUnknownFields: true}
	_protoBufferPool = sync.Pool{
		New: func() interface{} {
			return proto.NewBuffer(make([]byte, 1024))
		},
	}
)

type protoCodec struct {
	reqMessage proto.Message
}

func newProtoCodec(reqType func() proto.Message) *protoCodec {
	return &protoCodec{reqMessage: reqType()}
}

func (c *protoCodec) Decode(req *yarpc.Buffer) (interface{}, error) {
	if err := proto.Unmarshal(req.Bytes(), c.reqMessage); err != nil {
		return nil, err
	}
	return c.reqMessage, nil
}

func (c *protoCodec) Encode(res interface{}) (*yarpc.Buffer, error) {
	if res == nil {
		return &yarpc.Buffer{}, nil
	}
	if message, ok := res.(proto.Message); ok {
		if message == nil {
			return &yarpc.Buffer{}, nil
		}
		return marshalProto(message)
	}
	return nil, yarpcerror.InternalErrorf("tried to encode a non-proto.Message in protobuf codec")
}

type jsonCodec struct {
	reqType proto.Message
}

func newJSONCodec(reqType func() proto.Message) *jsonCodec {
	return &jsonCodec{reqType: reqType()}
}

func (c *jsonCodec) Decode(req *yarpc.Buffer) (interface{}, error) {
	if err := _jsonUnmarshaler.Unmarshal(req, c.reqType); err != nil {
		return nil, err
	}
	return c.reqType, nil
}

func (c *jsonCodec) Encode(res interface{}) (*yarpc.Buffer, error) {
	if res == nil {
		return &yarpc.Buffer{}, nil
	}
	if message, ok := res.(proto.Message); ok {
		if message == nil {
			return &yarpc.Buffer{}, nil
		}
		return marshalJSON(message)
	}
	return nil, yarpcerror.InternalErrorf("tried to encode a non-proto.Message in json codec")
}
