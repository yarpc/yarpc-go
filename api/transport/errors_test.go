package transport

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBadRequestError(t *testing.T) {
	err := errors.New("derp")
	err = InboundBadRequestError(err)
	assert.True(t, IsBadRequestError(err))
	assert.Equal(t, "BadRequest: derp", err.Error())
}
