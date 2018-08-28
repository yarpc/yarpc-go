package yarpcerror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapHandlerError(t *testing.T) {
	assert.Nil(t, WrapHandlerError(nil, "foo", "bar"))
	assert.Equal(t, CodeInvalidArgument, FromError(WrapHandlerError(Newf(CodeInvalidArgument, ""), "foo", "bar")).Code())
	assert.Equal(t, CodeUnknown, FromError(WrapHandlerError(errors.New(""), "foo", "bar")).Code())
}
