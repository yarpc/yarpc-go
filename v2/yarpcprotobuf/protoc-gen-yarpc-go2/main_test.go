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

package main

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"os"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Used to register the FileDescriptorSets with the protobuf registry.
	_ "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go2/internal/tests/gen/proto/src/common"
	_ "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go2/internal/tests/gen/proto/src/keyvalue"
	_ "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go2/internal/tests/gen/proto/src/stream"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		desc           string
		filename       string
		dependencies   []string
		output         string
		outputFilename string
		fixture        string
	}{
		{
			desc:           "empty service",
			filename:       "src/common/common.proto",
			output:         "internal/tests/gen/proto/src/common/common.pb.yarpc.go",
			outputFilename: "src/common/common.pb.yarpc.go",
			fixture:        "internal/tests/gen/proto/src/common/common.pb.yarpc.go.fixture",
		},
		{
			desc:           "unary service",
			filename:       "src/keyvalue/key_value.proto",
			dependencies:   []string{"src/common/common.proto"},
			output:         "internal/tests/gen/proto/src/keyvalue/key_value.pb.yarpc.go",
			outputFilename: "src/keyvalue/key_value.pb.yarpc.go",
			fixture:        "internal/tests/gen/proto/src/keyvalue/key_value.pb.yarpc.go.fixture",
		},
		{
			desc:           "stream service",
			filename:       "src/stream/stream.proto",
			dependencies:   []string{"src/common/common.proto"},
			output:         "internal/tests/gen/proto/src/stream/stream.pb.yarpc.go",
			outputFilename: "src/stream/stream.pb.yarpc.go",
			fixture:        "internal/tests/gen/proto/src/stream/stream.pb.yarpc.go.fixture",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			data := getCodeGeneratorRequestBytes(t, tt.filename, tt.dependencies...)
			input := bytes.NewReader(data)
			output := bytes.NewBuffer(nil)
			require.NoError(t, run(input, output))

			act := &plugin_go.CodeGeneratorResponse{}
			require.NoError(t, proto.Unmarshal(output.Bytes(), act))
			require.Len(t, act.File, 1)

			content, err := ioutil.ReadFile(tt.fixture)
			require.NoError(t, err)

			exp := &plugin_go.CodeGeneratorResponse{
				File: []*plugin_go.CodeGeneratorResponse_File{
					{
						Name:    proto.String(tt.outputFilename),
						Content: proto.String(string(content)),
					},
				},
			}
			// Controlled by a call to `make fixtures`.
			if os.Getenv("FIXTURES") != "" {
				newFixture := act.File[0]
				require.NotNil(t, newFixture.Content)
				require.NoError(t, ioutil.WriteFile(tt.fixture, []byte(*newFixture.Content), 0755))
				return
			}
			assert.Equal(t, exp, act)
		})
	}
}

func getCodeGeneratorRequestBytes(t *testing.T, filename string, dependencies ...string) []byte {
	req := &plugin_go.CodeGeneratorRequest{
		Parameter: proto.String(
			"Msrc/common/common.proto=go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go2/internal/tests/gen/proto/src/common",
		),
		FileToGenerate: []string{
			filename,
		},
		ProtoFile: []*descriptor.FileDescriptorProto{
			getFileDescriptorProto(t, filename),
		},
	}
	for _, d := range dependencies {
		req.ProtoFile = append(req.ProtoFile, getFileDescriptorProto(t, d))
	}

	bytes, err := proto.Marshal(req)
	require.NoError(t, err)

	return bytes
}

// getFileDescriptorProto unmarshals the given filename into a *FileDescriptorProto.
// Note that this process is NOT necessarily equivalent to running the plugins via
// protoc.
// In our case, the fixture reflection bytes will differ slightly than what is
// produced by protoc.
func getFileDescriptorProto(t *testing.T, filename string) *descriptor.FileDescriptorProto {
	fd := proto.FileDescriptor(filename)
	require.NotEmpty(t, fd)

	// The expected file descriptor set must be compressed.
	reader, err := gzip.NewReader(bytes.NewReader(fd))
	require.NoError(t, err)

	data, err := ioutil.ReadAll(reader)
	require.NoError(t, err)

	fdp := &descriptor.FileDescriptorProto{}
	require.NoError(t, proto.Unmarshal(data, fdp))

	return fdp
}
