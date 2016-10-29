package json

import (
	"context"
	"testing"

	"go.uber.org/yarpc"

	"github.com/stretchr/testify/assert"
)

func TestWrapHandlerInvalid(t *testing.T) {
	tests := []struct {
		Name string
		Func interface{}
	}{
		{"empty", func() {}},
		{
			"wrong-response",
			func(context.Context, yarpc.ReqMeta, map[string]interface{}) (yarpc.ResMeta, error) {
				return nil, nil
			},
		},
		{
			"wrong-request",
			func(context.Context, string, *struct{}) (*struct{}, yarpc.ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"wrong-req-meta",
			func(context.Context, yarpc.CallReqMeta, *struct{}) (*struct{}, yarpc.ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"wrong-res-meta",
			func(context.Context, yarpc.ReqMeta, *struct{}) (*struct{}, yarpc.CallResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"non-pointer-req",
			func(context.Context, yarpc.ReqMeta, struct{}) (*struct{}, yarpc.ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"non-pointer-res",
			func(context.Context, yarpc.ReqMeta, *struct{}) (struct{}, yarpc.ResMeta, error) {
				return struct{}{}, nil, nil
			},
		},
		{
			"non-string-key",
			func(context.Context, yarpc.ReqMeta, map[int32]interface{}) (*struct{}, yarpc.ResMeta, error) {
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
			func(context.Context, yarpc.ReqMeta, *struct{}) (*struct{}, yarpc.ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"bar",
			func(context.Context, yarpc.ReqMeta, map[string]interface{}) (*struct{}, yarpc.ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"baz",
			func(context.Context, yarpc.ReqMeta, map[string]interface{}) (map[string]interface{}, yarpc.ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"qux",
			func(context.Context, yarpc.ReqMeta, interface{}) (map[string]interface{}, yarpc.ResMeta, error) {
				return nil, nil, nil
			},
		},
	}

	for _, tt := range tests {
		wrapHandler(tt.Name, tt.Func)
	}
}
