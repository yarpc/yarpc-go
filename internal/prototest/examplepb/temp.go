package examplepb

import "github.com/golang/protobuf/proto"

func init() {

	proto.RegisterType((*GetValueRequest)(nil), "uber.yarpc.internal.examples.protobuf.example.GetValueRequest")

	proto.RegisterType((*GetValueResponse)(nil), "uber.yarpc.internal.examples.protobuf.example.GetValueResponse")

	proto.RegisterType((*SetValueRequest)(nil), "uber.yarpc.internal.examples.protobuf.example.SetValueRequest")

	proto.RegisterType((*SetValueResponse)(nil), "uber.yarpc.internal.examples.protobuf.example.SetValueResponse")

	proto.RegisterType((*EchoOutRequest)(nil), "uber.yarpc.internal.examples.protobuf.example.EchoOutRequest")

	proto.RegisterType((*EchoOutResponse)(nil), "uber.yarpc.internal.examples.protobuf.example.EchoOutResponse")

	proto.RegisterType((*EchoInRequest)(nil), "uber.yarpc.internal.examples.protobuf.example.EchoInRequest")

	proto.RegisterType((*EchoInResponse)(nil), "uber.yarpc.internal.examples.protobuf.example.EchoInResponse")

	proto.RegisterType((*EchoBothRequest)(nil), "uber.yarpc.internal.examples.protobuf.example.EchoBothRequest")

	proto.RegisterType((*EchoBothResponse)(nil), "uber.yarpc.internal.examples.protobuf.example.EchoBothResponse")

}

