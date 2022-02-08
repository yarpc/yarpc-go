package v2

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/runtime/protoiface"
	"google.golang.org/protobuf/runtime/protoimpl"
)

// ProtobufMessageV1 converts either a v1 or v2 message to a v1 message.
func ProtobufMessageV1(message proto.Message) protoiface.MessageV1{
	return protoimpl.X.ProtoMessageV1Of(message)
}
