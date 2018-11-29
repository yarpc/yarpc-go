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
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcerror"
	spb "google.golang.org/genproto/googleapis/rpc/status"
)

type protoCodec struct {
	reqMessage proto.Message
}

func newProtoCodec(reqMessage func() proto.Message) *protoCodec {
	return &protoCodec{reqMessage: reqMessage()}
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

func (c *protoCodec) EncodeError(err error) (*yarpc.Buffer, error) {
	if err == nil {
		return &yarpc.Buffer{}, nil
	}

	info := yarpcerror.ExtractInfo(err)
	p := spb.Status{
		Code:    int32(info.Code),
		Message: info.Message,
	}
	if details := yarpcerror.ExtractDetails(err); details != nil {
		if m, ok := details.(proto.Message); ok {
			any, err := ptypes.MarshalAny(m)
			if err != nil {
				return nil, err
			}
			p.Details = append(p.Details, any)
		} else if messages, ok := details.([]proto.Message); ok {
			for _, m := range messages {
				any, err := ptypes.MarshalAny(m)
				if err != nil {
					return nil, err
				}
				p.Details = append(p.Details, any)
			}
		} else {
			return nil, yarpcerror.InternalErrorf("tried to encode a non-proto.Message in proto error codec")
		}
	}
	return marshalProto(&p)
}

type jsonCodec struct {
	reqMessage proto.Message
}

func newJSONCodec(reqMessage func() proto.Message) *jsonCodec {
	return &jsonCodec{reqMessage: reqMessage()}
}

func (c *jsonCodec) Decode(req *yarpc.Buffer) (interface{}, error) {
	if err := _jsonUnmarshaler.Unmarshal(req, c.reqMessage); err != nil {
		return nil, err
	}
	return c.reqMessage, nil
}

func (c *jsonCodec) Encode(res interface{}) (*yarpc.Buffer, error) {
	return encodeAsJSON(res)
}

func encodeAsJSON(res interface{}) (*yarpc.Buffer, error) {
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

func (c *jsonCodec) EncodeError(err error) (*yarpc.Buffer, error) {
	// Here we only encode the proto.Message error details instead of
	// wrapping it in google/rpc/status.proto
	// Using google/rpc/status.proto it in a status is required for the
	// combination of grpc/proto, but undefined for proto error details
	// with other transports/encodings.
	details := yarpcerror.ExtractDetails(err)
	if details == nil {
		return nil, nil
	}
	return encodeAsJSON(details)
}
