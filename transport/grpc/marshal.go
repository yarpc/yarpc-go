package grpc

import (
	"github.com/gogo/googleapis/google/rpc"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/status"
)

func unmarshalError(body []byte) error {
	protobufStatus := &rpc.Status{}
	err := proto.Unmarshal(body, protobufStatus)
	if err != nil {
		return err
	}
	return status.ErrorProto(protobufStatus)
}

func marshalError(st *status.Status) []byte {
	if len(st.Details()) == 0 {
		return nil
	}
	buf, err := proto.Marshal(st.Proto())
	if err != nil {
		return nil
	}
	return buf
}
