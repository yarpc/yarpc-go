package grpc

import (
	"errors"
	"testing"

	"go.uber.org/yarpc/transport"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
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

func TestGetContextMetadata(t *testing.T) {
	type testStruct struct {
		ctx       context.Context
		wantMD    metadata.MD
		wantError error
	}
	tests := []testStruct{
		func() (s testStruct) {
			s.wantMD = metadata.MD{
				"key1": []string{"value1"},
				"key2": []string{"value2"},
			}
			s.ctx = metadata.NewContext(context.Background(), s.wantMD)
			s.wantError = nil
			return
		}(),
		func() (s testStruct) {
			s.ctx = context.Background()
			s.wantMD = nil
			s.wantError = errCantExtractMetadata{ctx: s.ctx}
			return
		}(),
	}

	for _, tt := range tests {
		md, err := getContextMetadata(tt.ctx)

		assert.Equal(t, tt.wantMD, md)
		assert.Equal(t, tt.wantError, err)
	}

}

func TestExtractMetadataHeader(t *testing.T) {
	type testStruct struct {
		headerName      string
		md              metadata.MD
		wantHeaderValue string
		wantError       error
	}
	tests := []testStruct{
		func() (s testStruct) {
			s.headerName = "key"
			s.wantHeaderValue = "value"
			s.md = metadata.MD{
				s.headerName: []string{s.wantHeaderValue},
			}
			s.wantError = nil
			return
		}(),
		func() (s testStruct) {
			s.headerName = "key"
			s.wantHeaderValue = ""
			s.md = metadata.MD{}
			s.wantError = errCantExtractHeader{MD: s.md, Name: s.headerName}
			return
		}(),
		func() (s testStruct) {
			s.headerName = "key"
			s.wantHeaderValue = ""
			s.md = metadata.MD{
				s.headerName: []string{},
			}
			s.wantError = errCantExtractHeader{MD: s.md, Name: s.headerName}
			return
		}(),
		func() (s testStruct) {
			s.headerName = "key"
			s.wantHeaderValue = ""
			s.md = metadata.MD{
				s.headerName: []string{"test1", "test2", "test3"},
			}
			s.wantError = errCantExtractHeader{MD: s.md, Name: s.headerName}
			return
		}(),
	}

	for _, tt := range tests {
		headerValue, err := extractMetadataHeader(tt.md, tt.headerName)

		assert.Equal(t, tt.wantError, err)
		assert.Equal(t, tt.wantHeaderValue, headerValue)
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
