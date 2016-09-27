package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yarpc/yarpc-go/transport"
	"google.golang.org/grpc/metadata"
)

func TestResponseWriter_Write(t *testing.T) {
	strMsg := "this is a test"
	byteMsg := []byte(strMsg)
	var r response
	rw := newResponseWriter(&r)

	changed, err := rw.Write(byteMsg)

	assert.Equal(t, len(byteMsg), changed)
	assert.Equal(t, error(nil), err)
	assert.Equal(t, strMsg, string(r.body.Bytes()))
}

func TestResponseWriter_AddHeaders(t *testing.T) {
	caller := "teeeeest"
	encoding := "raw"
	inputHeaders := transport.HeadersFromMap(map[string]string{
		CallerHeader:   caller,
		EncodingHeader: encoding,
	})
	expectedHeaders := metadata.New(map[string]string{
		ApplicationHeaderPrefix + CallerHeader:   caller,
		ApplicationHeaderPrefix + EncodingHeader: encoding,
	})

	var r response
	rw := newResponseWriter(&r)

	rw.AddHeaders(inputHeaders)

	assert.Equal(t, expectedHeaders, r.headers)
}

func TestResponseWriter_SetApplicationError(t *testing.T) {
	var r response
	rw := newResponseWriter(&r)

	rw.SetApplicationError()

	// No action on Application Error
}
