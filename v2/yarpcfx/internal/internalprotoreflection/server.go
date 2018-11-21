package internalprotoreflection

import (
	"io"

	"go.uber.org/yarpc/encoding/protobuf/reflection"
	"go.uber.org/yarpc/v2"
	rpb "go.uber.org/yarpc/v2/yarpcfx/internal/internalprotobuf/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	serviceNames        []string
	fileDescriptorIndex fileDescriptorIndex
}

// NewServer new yarpc meta procedures a dispatcher, exposing information about
// the dispatcher itself.
func NewServer(metas []reflection.ServerMeta) ([]yarpc.TransportProcedure, error) {
	server, err := newServerFromMetas(metas)
	if err != nil {
		return nil, err
	}
	return rpb.BuildServerReflectionYARPCProcedures(server), nil
}

func newServerFromMetas(metas []reflection.ServerMeta) (*server, error) {
	metas = append(metas, reflection.ServerMeta{
		ServiceName:     "grpc.reflection.v1alpha.ServerReflection",
		FileDescriptors: rpb.YARPCReflectionFileDescriptors,
	})

	names, index, err := indexReflectionMetas(metas)
	if err != nil {
		return nil, err
	}
	return &server{
		serviceNames:        names,
		fileDescriptorIndex: index,
	}, nil
}

func notFoundResponse(errMsg string) *rpb.ServerReflectionResponse_ErrorResponse {
	return &rpb.ServerReflectionResponse_ErrorResponse{
		ErrorResponse: &rpb.ErrorResponse{
			ErrorCode:    int32(codes.NotFound),
			ErrorMessage: errMsg,
		},
	}
}

func (s *server) handleFileByFileName(req *rpb.ServerReflectionRequest_FileByFilename, out *rpb.ServerReflectionResponse) {
	b, err := s.fileDescriptorIndex.descriptorForFile(req.FileByFilename)
	if err != nil {
		out.MessageResponse = notFoundResponse(err.Error())
		return
	}
	out.MessageResponse = &rpb.ServerReflectionResponse_FileDescriptorResponse{
		FileDescriptorResponse: &rpb.FileDescriptorResponse{FileDescriptorProto: [][]byte{b}},
	}
}

func (s *server) handleFileContainingSymbol(req *rpb.ServerReflectionRequest_FileContainingSymbol, out *rpb.ServerReflectionResponse) {
	b, err := s.fileDescriptorIndex.descriptorForSymbol(req.FileContainingSymbol)
	if err != nil {
		out.MessageResponse = notFoundResponse(err.Error())
		return
	}
	out.MessageResponse = &rpb.ServerReflectionResponse_FileDescriptorResponse{
		FileDescriptorResponse: &rpb.FileDescriptorResponse{FileDescriptorProto: [][]byte{b}},
	}
}

func (s *server) handleFileContainingExtension(req *rpb.ServerReflectionRequest_FileContainingExtension, out *rpb.ServerReflectionResponse) {
	typeName := req.FileContainingExtension.ContainingType
	extNum := req.FileContainingExtension.ExtensionNumber
	b, err := s.fileDescriptorIndex.descriptorForExtensionAndNumber(typeName, extNum)
	if err != nil {
		out.MessageResponse = notFoundResponse(err.Error())
		return
	}
	out.MessageResponse = &rpb.ServerReflectionResponse_FileDescriptorResponse{
		FileDescriptorResponse: &rpb.FileDescriptorResponse{FileDescriptorProto: [][]byte{b}},
	}
}

func (s *server) handleAllExtensionNumbersOfType(req *rpb.ServerReflectionRequest_AllExtensionNumbersOfType, out *rpb.ServerReflectionResponse) {
	extNums, err := s.fileDescriptorIndex.allExtensionNumbersForTypeName(req.AllExtensionNumbersOfType)
	if err != nil {
		out.MessageResponse = notFoundResponse(err.Error())
		return
	}
	out.MessageResponse = &rpb.ServerReflectionResponse_AllExtensionNumbersResponse{
		AllExtensionNumbersResponse: &rpb.ExtensionNumberResponse{
			BaseTypeName:    req.AllExtensionNumbersOfType,
			ExtensionNumber: extNums,
		},
	}
}

func (s *server) handleListServices(req *rpb.ServerReflectionRequest_ListServices, out *rpb.ServerReflectionResponse) {
	svcNames := s.serviceNames
	serviceResponses := make([]*rpb.ServiceResponse, len(svcNames))
	for i, n := range svcNames {
		serviceResponses[i] = &rpb.ServiceResponse{
			Name: n,
		}
	}
	out.MessageResponse = &rpb.ServerReflectionResponse_ListServicesResponse{
		ListServicesResponse: &rpb.ListServiceResponse{
			Service: serviceResponses,
		},
	}
}

// ServerReflectionInfo is the reflection service handler.
func (s *server) ServerReflectionInfo(stream rpb.ServerReflectionServiceServerReflectionInfoYARPCServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		out := &rpb.ServerReflectionResponse{
			ValidHost:       in.Host,
			OriginalRequest: in,
		}
		switch req := in.MessageRequest.(type) {
		case *rpb.ServerReflectionRequest_FileByFilename:
			s.handleFileByFileName(req, out)
		case *rpb.ServerReflectionRequest_FileContainingSymbol:
			s.handleFileContainingSymbol(req, out)
		case *rpb.ServerReflectionRequest_FileContainingExtension:
			s.handleFileContainingExtension(req, out)
		case *rpb.ServerReflectionRequest_AllExtensionNumbersOfType:
			s.handleAllExtensionNumbersOfType(req, out)
		case *rpb.ServerReflectionRequest_ListServices:
			s.handleListServices(req, out)
		default:
			return status.Errorf(codes.InvalidArgument, "invalid MessageRequest: %v", in.MessageRequest)
		}

		if err := stream.Send(out); err != nil {
			return err
		}
	}
}
