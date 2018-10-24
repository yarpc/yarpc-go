package protobuf

import (
	"encoding/base64"

	"github.com/gogo/protobuf/proto"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
)

const _grpcStatusDetailsHeaderKey = "grpc-status-details-bin"

// ErrorDetailsFromHeaders pulls and decodes error details from the passed in
// headers.
func ErrorDetailsFromHeaders(headers map[string]string) []interface{} {
	statusDetailsBinary := headers[_grpcStatusDetailsHeaderKey]

	decodedHeader, err := decodeBinaryHeader(statusDetailsBinary)
	if err != nil {
		return nil
	}

	s := &spb.Status{}
	if err := proto.Unmarshal(decodedHeader, s); err != nil {
		return nil
	}
	return status.FromProto(s).Details()
}

func decodeBinaryHeader(value string) ([]byte, error) {
	isInputPadded := len(value)%4 == 0
	if isInputPadded {
		return base64.StdEncoding.DecodeString(value)
	}
	return base64.RawStdEncoding.DecodeString(value)
}
