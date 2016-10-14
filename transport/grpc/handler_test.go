package grpc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetServiceAndProcedureFromMethod(t *testing.T) {
	tests := []struct {
		method        string
		wantService   string
		wantProcedure string
		wantError     error
	}{
		{
			method:        "foo/bar",
			wantService:   "foo",
			wantProcedure: "bar",
		},
		{
			method:        "/foo/bar",
			wantService:   "foo",
			wantProcedure: "bar",
		},
		{
			method:        "/foo%2Fla/bar%2Fmoo",
			wantService:   "foo/la",
			wantProcedure: "bar/moo",
		},
		{
			method:    "",
			wantError: errors.New("no service procedure provided"),
		},
		{
			method:    "foo%/bar",
			wantError: errors.New("could not parse service for request: foo%, error: invalid URL escape \"%\""),
		},
		{
			method:    "foo/bar%",
			wantError: errors.New("could not parse procedure for request: bar%, error: invalid URL escape \"%\""),
		},
	}

	for _, tt := range tests {
		service, procedure, err := getServiceAndProcedureFromMethod(tt.method)

		assert.Equal(t, tt.wantError, err)
		assert.Equal(t, tt.wantService, service)
		assert.Equal(t, tt.wantProcedure, procedure)
	}
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
