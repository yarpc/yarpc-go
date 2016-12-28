package encoding

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNilCall(t *testing.T) {
	call := CallFromContext(context.Background())
	require.Nil(t, call)

	assert.Equal(t, "", call.Caller())
	assert.Equal(t, "", call.Service())
	assert.Equal(t, "", string(call.Encoding()))
	assert.Equal(t, "", call.Procedure())
	assert.Equal(t, "", call.ShardKey())
	assert.Equal(t, "", call.RoutingKey())
	assert.Equal(t, "", call.RoutingDelegate())
	assert.Equal(t, "", call.Header("foo"))
	assert.Empty(t, call.HeaderNames())

	assert.Error(t, call.WriteResponseHeader("foo", "bar"))
}
