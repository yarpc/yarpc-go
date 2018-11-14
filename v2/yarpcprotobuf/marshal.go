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
	"io"

	"github.com/gogo/protobuf/proto"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/internal/internalbufferpool"
	"go.uber.org/yarpc/v2/yarpcerror"
	"go.uber.org/yarpc/v2/yarpcjson"
)

func unmarshal(encoding yarpc.Encoding, reader io.Reader, message proto.Message) error {
	buf := internalbufferpool.Get()
	defer internalbufferpool.Put(buf)
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
		return _jsonUnmarshaler.Unmarshal(reader, message)
	default:
		return yarpcerror.Newf(yarpcerror.CodeInternal, "failed to unmarshal unexpected encoding %q", encoding)
	}
}

func marshal(encoding yarpc.Encoding, message proto.Message) (*yarpc.Buffer, error) {
	switch encoding {
	case Encoding:
		return marshalProto(message)
	case yarpcjson.Encoding:
		return marshalJSON(message)
	default:
		return nil, yarpcerror.Newf(yarpcerror.CodeInternal, "failed to marshal unexpected encoding %q", encoding)
	}
}

func marshalProto(message proto.Message) (*yarpc.Buffer, error) {
	buf, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}
	return yarpc.NewBufferBytes(buf), nil
}

func marshalJSON(message proto.Message) (*yarpc.Buffer, error) {
	buf := &yarpc.Buffer{}
	if err := _jsonMarshaler.Marshal(buf, message); err != nil {
		return nil, err
	}
	return buf, nil
}
