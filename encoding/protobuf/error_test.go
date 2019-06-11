package protobuf

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/yarpc/yarpcerrors"
)

func TestNewOK(t *testing.T) {
	err := NewError(yarpcerrors.CodeOK, "okay")
	assert.Nil(t, err)

	assert.Equal(t, GetErrorCode(err), yarpcerrors.CodeOK)
	assert.Equal(t, GetErrorMessage(err), "")
}

func TestNew(t *testing.T) {
	err := NewError(yarpcerrors.CodeNotFound, "unfounded accusation")
	assert.Equal(t, GetErrorCode(err), yarpcerrors.CodeNotFound)
	assert.Equal(t, GetErrorMessage(err), "unfounded accusation")
	assert.Contains(t, err.Error(), "unfounded accusation")
}

func TestForeignError(t *testing.T) {
	err := errors.New("to err is go")
	assert.Equal(t, GetErrorCode(err), yarpcerrors.CodeUnknown)
	assert.Equal(t, GetErrorMessage(err), "to err is go")
}
