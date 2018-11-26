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

package internalprotoreflection

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/v2"
	rpb "go.uber.org/yarpc/v2/yarpcfx/internal/internalprotoreflection/grpc_reflection_v1alpha"
	"go.uber.org/yarpc/v2/yarpcprotobuf/reflection"
)

func TestNewServers(t *testing.T) {
	tests := []struct {
		name         string
		args         []reflection.ServerMeta
		wantServices []string
		wantErr      string
	}{
		{
			name:         "none",
			args:         []reflection.ServerMeta{},
			wantServices: []string{"grpc.reflection.v1alpha.ServerReflection"},
		},
		{
			name: "invalid fd",
			args: []reflection.ServerMeta{
				{ServiceName: "yaha", FileDescriptors: [][]byte{{0x00}}},
			},
			wantErr: "bad gzipped descriptor",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transports, terr := NewServer(tt.args)
			server, err := newServerFromMetas(tt.args)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				assert.NotNil(t, transports)

				assert.NoError(t, terr)
				for _, s := range tt.wantServices {
					assert.Contains(t, server.serviceNames, s)
					fd, err := server.fileDescriptorIndex.descriptorForSymbol(s)
					assert.NoError(t, err)
					assert.NotNil(t, fd)
				}
			} else {
				assert.Error(t, terr)
				assert.Nil(t, transports)

				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

type errorClient struct {
	sendErr      error
	sendRequest  *rpb.ServerReflectionRequest
	errOnReceive error
}

func (m *errorClient) Context() context.Context {
	return context.Background()
}

func (m *errorClient) Recv(...yarpc.StreamOption) (*rpb.ServerReflectionRequest, error) {
	return m.sendRequest, m.sendErr
}

func (m *errorClient) Send(response *rpb.ServerReflectionResponse, options ...yarpc.StreamOption) error {
	return m.errOnReceive
}

func TestErrors(t *testing.T) {
	s := &server{}
	t.Run("request nil", func(t *testing.T) {
		err := s.ServerReflectionInfo(&errorClient{
			sendErr: io.EOF,
		})
		require.NoError(t, err)
	})
	t.Run("send error", func(t *testing.T) {
		errToSend := fmt.Errorf("send this")
		err := s.ServerReflectionInfo(&errorClient{
			sendErr: errToSend,
		})
		require.EqualError(t, err, errToSend.Error())
	})
	t.Run("invalid request", func(t *testing.T) {
		err := s.ServerReflectionInfo(&errorClient{
			sendRequest: &rpb.ServerReflectionRequest{},
		})
		require.EqualError(t, err, "rpc error: code = InvalidArgument desc = invalid MessageRequest: <nil>")
	})
	t.Run("simulate receive error", func(t *testing.T) {
		errOnReceive := fmt.Errorf("rec this")
		err := s.ServerReflectionInfo(&errorClient{
			sendRequest:  &rpb.ServerReflectionRequest{MessageRequest: &rpb.ServerReflectionRequest_ListServices{ListServices: "yes"}},
			errOnReceive: errOnReceive,
		})
		require.EqualError(t, err, errOnReceive.Error())
	})
}

type mockClient struct {
	requests  []*rpb.ServerReflectionRequest
	responses []*rpb.ServerReflectionResponse
}

func (m *mockClient) Context() context.Context {
	return context.Background()
}

func (m *mockClient) Recv(...yarpc.StreamOption) (*rpb.ServerReflectionRequest, error) {
	if len(m.requests) == 0 {
		return nil, io.EOF
	}
	r := m.requests[0]
	m.requests = m.requests[1:]
	return r, nil
}

func (m *mockClient) Send(response *rpb.ServerReflectionResponse, options ...yarpc.StreamOption) error {
	response.OriginalRequest = nil
	m.responses = append(m.responses, response)
	return nil
}

func filedescriptorResponseWithContent(bytes []byte) *rpb.ServerReflectionResponse {
	return &rpb.ServerReflectionResponse{
		MessageResponse: &rpb.ServerReflectionResponse_FileDescriptorResponse{
			FileDescriptorResponse: &rpb.FileDescriptorResponse{FileDescriptorProto: [][]byte{bytes}},
		},
	}
}

func extensionsResponse(basename string, extensions ...int32) *rpb.ServerReflectionResponse {
	return &rpb.ServerReflectionResponse{
		MessageResponse: &rpb.ServerReflectionResponse_AllExtensionNumbersResponse{
			AllExtensionNumbersResponse: &rpb.ExtensionNumberResponse{
				BaseTypeName:    basename,
				ExtensionNumber: extensions,
			},
		},
	}
}

func TestRequestResponse(t *testing.T) {
	mockIndex := &descriptorIndex{
		descriptorsByFile:   map[string][]byte{"file.proto": {0xff, 0xf0}},
		descriptorsBySymbol: map[string][]byte{"some.symbol": {0xff}},
		descriptorsByExtensionAndNumber: map[string]map[int32][]byte{
			"some.symbol": {
				42: {0x1f},
				21: {0x1f},
			},
		},
	}
	s := &server{
		serviceNames:        []string{"test"},
		fileDescriptorIndex: mockIndex,
	}
	tests := []struct {
		name              string
		requests          []*rpb.ServerReflectionRequest
		expectedResponses []*rpb.ServerReflectionResponse
	}{
		{
			name: "list services",
			requests: []*rpb.ServerReflectionRequest{
				{MessageRequest: &rpb.ServerReflectionRequest_ListServices{ListServices: "yes"}},
			},
			expectedResponses: []*rpb.ServerReflectionResponse{
				{MessageResponse: &rpb.ServerReflectionResponse_ListServicesResponse{ListServicesResponse: &rpb.ListServiceResponse{Service: []*rpb.ServiceResponse{{Name: "test"}}}}},
			},
		}, {
			name: "by filename",
			requests: []*rpb.ServerReflectionRequest{
				{MessageRequest: &rpb.ServerReflectionRequest_FileByFilename{FileByFilename: "file.proto"}},
				{MessageRequest: &rpb.ServerReflectionRequest_FileByFilename{FileByFilename: "error"}},
			},
			expectedResponses: []*rpb.ServerReflectionResponse{
				filedescriptorResponseWithContent(mockIndex.descriptorsByFile["file.proto"]),
				{MessageResponse: notFoundResponse(`could not find descriptor for file "error"`)},
			},
		}, {
			name: "by symbol",
			requests: []*rpb.ServerReflectionRequest{
				{MessageRequest: &rpb.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: "some.symbol"}},
				{MessageRequest: &rpb.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: "error"}},
			},
			expectedResponses: []*rpb.ServerReflectionResponse{
				filedescriptorResponseWithContent(mockIndex.descriptorsBySymbol["some.symbol"]),
				{MessageResponse: notFoundResponse(`could not find descriptor for symbol "error"`)},
			},
		}, {
			name: "by extension",
			requests: []*rpb.ServerReflectionRequest{
				{MessageRequest: &rpb.ServerReflectionRequest_FileContainingExtension{FileContainingExtension: &rpb.ExtensionRequest{ContainingType: "some.symbol", ExtensionNumber: 42}}},
				{MessageRequest: &rpb.ServerReflectionRequest_FileContainingExtension{FileContainingExtension: &rpb.ExtensionRequest{ContainingType: "some.symbol", ExtensionNumber: 21}}},
				{MessageRequest: &rpb.ServerReflectionRequest_FileContainingExtension{FileContainingExtension: &rpb.ExtensionRequest{ContainingType: "some.symbol", ExtensionNumber: 0}}},
				{MessageRequest: &rpb.ServerReflectionRequest_FileContainingExtension{FileContainingExtension: &rpb.ExtensionRequest{ContainingType: "error", ExtensionNumber: -1}}},
			},
			expectedResponses: []*rpb.ServerReflectionResponse{
				filedescriptorResponseWithContent(mockIndex.descriptorsByExtensionAndNumber["some.symbol"][42]),
				filedescriptorResponseWithContent(mockIndex.descriptorsByExtensionAndNumber["some.symbol"][21]),
				{MessageResponse: notFoundResponse(`could not find extension number 0 on type "some.symbol"`)},
				{MessageResponse: notFoundResponse(`could not find extension type "error"`)},
			},
		}, {
			name: "by extension",
			requests: []*rpb.ServerReflectionRequest{
				{MessageRequest: &rpb.ServerReflectionRequest_AllExtensionNumbersOfType{AllExtensionNumbersOfType: "some.symbol"}},
				{MessageRequest: &rpb.ServerReflectionRequest_AllExtensionNumbersOfType{AllExtensionNumbersOfType: "error"}},
			},
			expectedResponses: []*rpb.ServerReflectionResponse{
				extensionsResponse("some.symbol", 42, 21),
				{MessageResponse: notFoundResponse(`could not find extension type "error"`)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockClient{requests: tt.requests}
			err := s.ServerReflectionInfo(mockClient)
			assert.NoError(t, err)
			for i, expected := range tt.expectedResponses {
				require.NotNil(t, mockClient.responses[i])
				assert.Equal(t, expected, mockClient.responses[i])
			}
		})
	}
}
