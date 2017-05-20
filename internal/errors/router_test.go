package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnrecognizedProcedureError(t *testing.T) {
	err := RouterUnrecognizedProcedureError("curly", "echo").(unrecognizedProcedureError)
	assert.Equal(t, `BadRequest: unrecognized procedure "echo" for service "curly"`, err.AsHandlerError().Error())
}

func TestUnrecognizedEncodingError(t *testing.T) {
	err := RouterUnrecognizedEncodingError([]string{"json", "thrift"}, "raw").(unrecognizedEncodingError)
	assert.Equal(t, `BadRequest: expected encoding "json" or "thrift" but got "raw"`, err.AsHandlerError().Error())
}
