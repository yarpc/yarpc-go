package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPassThroughCodecMarshal(t *testing.T) {
	codec := PassThroughCodec{}
	strMsg := "this is a test"
	byteSender := []byte(strMsg)

	result, err := codec.Marshal(&byteSender)

	assert.Equal(t, strMsg, string(result))
	assert.Equal(t, error(nil), err)
}

func TestPassThroughCodecMarshalError(t *testing.T) {
	codec := PassThroughCodec{}
	strMsg := "this is a test"

	result, err := codec.Marshal(&strMsg)

	assert.Equal(t, []byte(nil), result)
	assert.Equal(t, "expected sender of type *[]byte but got *string", err.Error())
}

func TestPassThroughCodecUnmarshal(t *testing.T) {
	codec := PassThroughCodec{}
	strMsg := "this is a test"
	data := []byte(strMsg)
	var receiver []byte

	err := codec.Unmarshal(data, &receiver)

	assert.Equal(t, strMsg, string(receiver))
	assert.Equal(t, error(nil), err)
}

func TestPassThroughCodecUnmarshalError(t *testing.T) {
	codec := PassThroughCodec{}
	strMsg := "this is a test"
	data := []byte(strMsg)
	var receiver string

	err := codec.Unmarshal(data, &receiver)

	assert.Equal(t, "", receiver)
	assert.Equal(t, "expected receiver of type *[]byte but got *string", err.Error())
}

func TestPassThroughCodecString(t *testing.T) {
	codec := PassThroughCodec{}

	assert.Equal(t, "raw", codec.String())
}
