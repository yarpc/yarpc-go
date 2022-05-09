// Copyright (c) 2022 Uber Technologies, Inc.
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

package protopluginv2

import (
	"testing"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestGoPackageName(t *testing.T) {
	tests := []struct {
		msg           string
		protoFileName *string
		protoPackage  *string
		goPackage     *string
		expectedName  string
	}{
		{
			msg:          "short go_package",
			goPackage:    proto.String("package"),
			expectedName: "package",
		},
		{
			msg:          "full go_package",
			goPackage:    proto.String("path/to/package"),
			expectedName: "package",
		},
		{
			msg:          "full go_package with alias",
			goPackage:    proto.String("path/to/package;alias"),
			expectedName: "alias",
		},
		{
			msg:          "proto package",
			protoPackage: proto.String("proto.package"),
			expectedName: "proto_package",
		},
		{
			msg:           "proto file name",
			protoFileName: proto.String("path/filename.proto"),
			expectedName:  "filename",
		},
		{
			msg:          "prefer go_package over proto package",
			goPackage:    proto.String("path/to/package1"),
			protoPackage: proto.String("proto.package2"),
			expectedName: "package1",
		},
		{
			msg:           "prefer proto package over proto file name",
			protoPackage:  proto.String("proto.package"),
			protoFileName: proto.String("filename.proto"),
			expectedName:  "proto_package",
		},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			proto := &descriptor.FileDescriptorProto{
				Name:    test.protoFileName,
				Package: test.protoPackage,
				Options: &descriptor.FileOptions{
					GoPackage: test.goPackage,
				},
			}
			assert.Equal(t, test.expectedName, defaultGoPackageName(proto))
		})
	}
}
