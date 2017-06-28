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

package testing

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/encoding/protobuf/protoc-gen-yarpc-go/internal/lib"
	_ "go.uber.org/yarpc/yarpcproto" // needed for proto.RegisterFile for Oneway type
)

func TestGolden(t *testing.T) {
	codeGeneratorRequest := &plugin_go.CodeGeneratorRequest{
		Parameter: proto.String("Myarpcproto/yarpc.proto=go.uber.org/yarpc/yarpcproto"),
		FileToGenerate: []string{
			"encoding/protobuf/protoc-gen-yarpc-go/internal/testing/testing.proto",
		},
		ProtoFile: []*descriptor.FileDescriptorProto{
			getFileDescriptorProto(t, "yarpcproto/yarpc.proto"),
			getFileDescriptorProto(t, "encoding/protobuf/protoc-gen-yarpc-go/internal/testing/testing.proto"),
		},
	}

	content, err := ioutil.ReadFile("testing.pb.yarpc.go.golden")
	require.NoError(t, err)
	expectedCodeGeneratorResponse := &plugin_go.CodeGeneratorResponse{
		File: []*plugin_go.CodeGeneratorResponse_File{
			{
				Name:    proto.String("encoding/protobuf/protoc-gen-yarpc-go/internal/testing/testing.pb.yarpc.go"),
				Content: proto.String(string(content)),
			},
		},
	}

	require.Equal(t, expectedCodeGeneratorResponse, lib.Runner.Run(codeGeneratorRequest))
}

func getFileDescriptorProto(t *testing.T, name string) *descriptor.FileDescriptorProto {
	fileDescriptor := proto.FileDescriptor(name)
	require.NotEmpty(t, fileDescriptor)
	reader, err := gzip.NewReader(bytes.NewReader(fileDescriptor))
	require.NoError(t, err)
	data, err := ioutil.ReadAll(reader)
	require.NoError(t, err)
	fileDescriptorProto := &descriptor.FileDescriptorProto{}
	require.NoError(t, proto.Unmarshal(data, fileDescriptorProto))
	return fileDescriptorProto
}
