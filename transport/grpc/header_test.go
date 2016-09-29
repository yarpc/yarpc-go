package grpc

import (
	"testing"

	"go.uber.org/yarpc/transport"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestHeaderMapper_ToGRPCMetadata(t *testing.T) {
	prefix := "test-"
	testMapper := headerMapper{prefix}

	inputHeaders := transport.HeadersFromMap(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})
	expectedMetadata := metadata.New(map[string]string{
		"key":           "value",
		prefix + "key1": "value1",
		prefix + "key2": "value2",
	})

	md := metadata.New(map[string]string{
		"key": "value",
	})
	actualMetadata := testMapper.ToGRPCMetadata(inputHeaders, md)

	assert.Equal(t, expectedMetadata, actualMetadata)
}

func TestHeaderMapper_ToGRPCMetadata_fromNil(t *testing.T) {
	prefix := "test-"
	testMapper := headerMapper{prefix}

	inputHeaders := transport.HeadersFromMap(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})
	expectedMetadata := metadata.New(map[string]string{
		prefix + "key1": "value1",
		prefix + "key2": "value2",
	})

	actualMetadata := testMapper.ToGRPCMetadata(inputHeaders, nil)

	assert.Equal(t, expectedMetadata, actualMetadata)
}

func TestHeaderMapper_FromGRPCMetadata(t *testing.T) {
	prefix := "test-"
	testMapper := headerMapper{prefix}

	inputMetadata := metadata.New(map[string]string{
		"key":           "value",
		prefix + "key1": "value1",
		prefix + "key2": "value2",
	})
	expectedHeaders := transport.HeadersFromMap(map[string]string{
		"key1": "value1",
		"key2": "value2",
	})

	actualHeaders := testMapper.FromGRPCMetadata(inputMetadata, transport.Headers{})

	assert.Equal(t, expectedHeaders, actualHeaders)
}

func TestHeaders_Add(t *testing.T) {
	headers := Headers{}
	headers["key1"] = []string{"value1"}
	headers["key2"] = []string{"value2"}

	headers.Add("testkey", "testvalue")

	assert.Equal(t, "value1", headers["key1"][0])
	assert.Equal(t, "value2", headers["key2"][0])
	assert.Equal(t, "testvalue", headers["testkey"][0])
}

func TestHeaders_Del(t *testing.T) {
	headers := Headers{}
	headers["key1"] = []string{"value1"}
	headers["key2"] = []string{"value2"}

	headers.Del("key2")

	assert.Equal(t, 1, len(headers))
	assert.Equal(t, "value1", headers["key1"][0])
	assert.Equal(t, []string(nil), headers["key2"])
}

func TestHeaders_Get(t *testing.T) {
	headers := Headers{}
	headers["key1"] = []string{"value1"}
	headers["key2"] = []string{"value2"}

	value := headers.Get("key2")

	assert.Equal(t, "value2", value)
}

func TestHeaders_Get_Nil(t *testing.T) {
	headers := Headers(nil)

	value := headers.Get("key2")

	assert.Equal(t, "", value)
}

func TestHeaders_Get_EmptyList(t *testing.T) {
	headers := Headers{}
	headers["key1"] = []string{}

	value := headers.Get("key1")

	assert.Equal(t, "", value)
}

func TestHeaders_Set(t *testing.T) {
	headers := Headers{}
	headers["key1"] = []string{"value1"}
	headers["key2"] = []string{"value2"}

	headers.Set("key2", "newValue")

	assert.Equal(t, []string{"newValue"}, headers["key2"])
}
