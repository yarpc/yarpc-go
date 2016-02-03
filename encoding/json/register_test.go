package json

import (
	"testing"

	"github.com/yarpc/yarpc-go"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestWrapHandlerInvalid(t *testing.T) {
	tests := []struct {
		Name string
		Func interface{}
	}{
		{"empty", func() {}},
		{
			"wrong-response",
			func(context.Context, yarpc.Meta, map[string]interface{}) (yarpc.Meta, error) {
				return nil, nil
			},
		},
		{
			"wrong-context",
			func(string, yarpc.Meta, *struct{}) (*struct{}, yarpc.Meta, error) {
				return nil, nil, nil
			},
		},
		{
			"wrong-meta",
			func(context.Context, string, *struct{}) (*struct{}, yarpc.Meta, error) {
				return nil, nil, nil
			},
		},
		{
			"non-pointer-req",
			func(context.Context, yarpc.Meta, struct{}) (*struct{}, yarpc.Meta, error) {
				return nil, nil, nil
			},
		},
		{
			"non-pointer-res",
			func(context.Context, yarpc.Meta, *struct{}) (struct{}, yarpc.Meta, error) {
				return struct{}{}, nil, nil
			},
		},
		{
			"non-string-key",
			func(context.Context, yarpc.Meta, map[int32]interface{}) (*struct{}, yarpc.Meta, error) {
				return nil, nil, nil
			},
		},
	}

	for _, tt := range tests {
		assert.Panics(t, assert.PanicTestFunc(func() {
			wrapHandler(tt.Name, tt.Func)
		}))
	}
}

func TestWrapHandlerValid(t *testing.T) {
	tests := []struct {
		Name string
		Func interface{}
	}{
		{
			"foo",
			func(context.Context, yarpc.Meta, *struct{}) (*struct{}, yarpc.Meta, error) {
				return nil, nil, nil
			},
		},
		{
			"bar",
			func(context.Context, yarpc.Meta, map[string]interface{}) (*struct{}, yarpc.Meta, error) {
				return nil, nil, nil
			},
		},
		{
			"baz",
			func(context.Context, yarpc.Meta, map[string]interface{}) (map[string]interface{}, yarpc.Meta, error) {
				return nil, nil, nil
			},
		},
	}

	for _, tt := range tests {
		wrapHandler(tt.Name, tt.Func)
	}
}
