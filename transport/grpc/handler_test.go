package grpc

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetServiceAndProcedureFromMethod_Base(t *testing.T) {
	method := "foo/bar"

	service, procedure, err := getServiceAndProcedureFromMethod(method)

	assert.Equal(t, nil, err)
	assert.Equal(t, "foo", service)
	assert.Equal(t, "bar", procedure)
}

func TestGetServiceAndProcedureFromMethod_ExtraSlash(t *testing.T) {
	method := "/foo/bar"

	service, procedure, err := getServiceAndProcedureFromMethod(method)

	assert.Equal(t, nil, err)
	assert.Equal(t, "foo", service)
	assert.Equal(t, "bar", procedure)
}

func TestGetServiceAndProcedureFromMethod_URLEncoded(t *testing.T) {
	expectedService := "foo/la"
	expectedProcedure := "bar/moo"
	method := fmt.Sprintf("/%s/%s", url.QueryEscape(expectedService), url.QueryEscape(expectedProcedure))

	service, procedure, err := getServiceAndProcedureFromMethod(method)

	assert.Equal(t, nil, err)
	assert.Equal(t, expectedService, service)
	assert.Equal(t, expectedProcedure, procedure)
}

func TestGetServiceAndProcedureFromMethod_ServiceDecodeError(t *testing.T) {
	invalidService := "foo%"
	method := fmt.Sprintf("/%s/bar", invalidService)

	_, _, err := getServiceAndProcedureFromMethod(method)

	assert.NotNil(t, err, "Invalid service encoding did not cause an error")
}

func TestGetServiceAndProcedureFromMethod_ProcedureDecodeError(t *testing.T) {
	invalidProcedure := "bar%"
	method := fmt.Sprintf("/foo/%s", invalidProcedure)

	_, _, err := getServiceAndProcedureFromMethod(method)

	assert.NotNil(t, err, "Invalid procedure encoding did not cause an error")
}

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
