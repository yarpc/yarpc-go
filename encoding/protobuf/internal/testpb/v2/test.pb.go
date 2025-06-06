// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v3.15.0
// source: encoding/protobuf/internal/testpb/v2/test.proto

package testpb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type TestMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Value string `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *TestMessage) Reset() {
	*x = TestMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_encoding_protobuf_internal_testpb_v2_test_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestMessage) ProtoMessage() {}

func (x *TestMessage) ProtoReflect() protoreflect.Message {
	mi := &file_encoding_protobuf_internal_testpb_v2_test_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestMessage.ProtoReflect.Descriptor instead.
func (*TestMessage) Descriptor() ([]byte, []int) {
	return file_encoding_protobuf_internal_testpb_v2_test_proto_rawDescGZIP(), []int{0}
}

func (x *TestMessage) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

var File_encoding_protobuf_internal_testpb_v2_test_proto protoreflect.FileDescriptor

var file_encoding_protobuf_internal_testpb_v2_test_proto_rawDesc = []byte{
	0x0a, 0x2f, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x65, 0x73,
	0x74, 0x70, 0x62, 0x2f, 0x76, 0x32, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x1c, 0x75, 0x62, 0x65, 0x72, 0x2e, 0x79, 0x61, 0x72, 0x70, 0x63, 0x2e, 0x65, 0x6e,
	0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x22,
	0x23, 0x0a, 0x0b, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x32, 0xc9, 0x01, 0x0a, 0x04, 0x54, 0x65, 0x73, 0x74, 0x12, 0x5d, 0x0a,
	0x05, 0x55, 0x6e, 0x61, 0x72, 0x79, 0x12, 0x29, 0x2e, 0x75, 0x62, 0x65, 0x72, 0x2e, 0x79, 0x61,
	0x72, 0x70, 0x63, 0x2e, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x1a, 0x29, 0x2e, 0x75, 0x62, 0x65, 0x72, 0x2e, 0x79, 0x61, 0x72, 0x70, 0x63, 0x2e, 0x65,
	0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x62, 0x0a, 0x06,
	0x44, 0x75, 0x70, 0x6c, 0x65, 0x78, 0x12, 0x29, 0x2e, 0x75, 0x62, 0x65, 0x72, 0x2e, 0x79, 0x61,
	0x72, 0x70, 0x63, 0x2e, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x1a, 0x29, 0x2e, 0x75, 0x62, 0x65, 0x72, 0x2e, 0x79, 0x61, 0x72, 0x70, 0x63, 0x2e, 0x65,
	0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x28, 0x01, 0x30, 0x01,
	0x42, 0x2d, 0x5a, 0x2b, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x69, 0x6e, 0x67, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74,
	0x65, 0x73, 0x74, 0x70, 0x62, 0x2f, 0x76, 0x32, 0x3b, 0x74, 0x65, 0x73, 0x74, 0x70, 0x62, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_encoding_protobuf_internal_testpb_v2_test_proto_rawDescOnce sync.Once
	file_encoding_protobuf_internal_testpb_v2_test_proto_rawDescData = file_encoding_protobuf_internal_testpb_v2_test_proto_rawDesc
)

func file_encoding_protobuf_internal_testpb_v2_test_proto_rawDescGZIP() []byte {
	file_encoding_protobuf_internal_testpb_v2_test_proto_rawDescOnce.Do(func() {
		file_encoding_protobuf_internal_testpb_v2_test_proto_rawDescData = protoimpl.X.CompressGZIP(file_encoding_protobuf_internal_testpb_v2_test_proto_rawDescData)
	})
	return file_encoding_protobuf_internal_testpb_v2_test_proto_rawDescData
}

var file_encoding_protobuf_internal_testpb_v2_test_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_encoding_protobuf_internal_testpb_v2_test_proto_goTypes = []any{
	(*TestMessage)(nil), // 0: uber.yarpc.encoding.protobuf.TestMessage
}
var file_encoding_protobuf_internal_testpb_v2_test_proto_depIdxs = []int32{
	0, // 0: uber.yarpc.encoding.protobuf.Test.Unary:input_type -> uber.yarpc.encoding.protobuf.TestMessage
	0, // 1: uber.yarpc.encoding.protobuf.Test.Duplex:input_type -> uber.yarpc.encoding.protobuf.TestMessage
	0, // 2: uber.yarpc.encoding.protobuf.Test.Unary:output_type -> uber.yarpc.encoding.protobuf.TestMessage
	0, // 3: uber.yarpc.encoding.protobuf.Test.Duplex:output_type -> uber.yarpc.encoding.protobuf.TestMessage
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_encoding_protobuf_internal_testpb_v2_test_proto_init() }
func file_encoding_protobuf_internal_testpb_v2_test_proto_init() {
	if File_encoding_protobuf_internal_testpb_v2_test_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_encoding_protobuf_internal_testpb_v2_test_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*TestMessage); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_encoding_protobuf_internal_testpb_v2_test_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_encoding_protobuf_internal_testpb_v2_test_proto_goTypes,
		DependencyIndexes: file_encoding_protobuf_internal_testpb_v2_test_proto_depIdxs,
		MessageInfos:      file_encoding_protobuf_internal_testpb_v2_test_proto_msgTypes,
	}.Build()
	File_encoding_protobuf_internal_testpb_v2_test_proto = out.File
	file_encoding_protobuf_internal_testpb_v2_test_proto_rawDesc = nil
	file_encoding_protobuf_internal_testpb_v2_test_proto_goTypes = nil
	file_encoding_protobuf_internal_testpb_v2_test_proto_depIdxs = nil
}
