package json

import (
	"context"
	"testing"

	"go.uber.org/yarpc"

	"github.com/stretchr/testify/assert"
)

func TestWrapUnaryHandlerInvalid(t *testing.T) {
	tests := []struct {
		Name string
		Func interface{}
	}{
		{"empty", func() {}},
		{"not-a-function", 0},
		{
			"wrong-args-in",
			func(context.Context, yarpc.ReqMeta) (*struct{}, yarpc.ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"wrong-ctx",
			func(string, yarpc.ReqMeta, *struct{}) (*struct{}, yarpc.ResMeta, error) {
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
			"wrong-req-body",
			func(context.Context, string, int) (*struct{}, yarpc.ResMeta, error) {
				return nil, nil, nil
			},
		},
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
			wrapUnaryHandler(tt.Name, tt.Func)
		}))
	}
}

func TestWrapUnaryHandlerValid(t *testing.T) {
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
		wrapUnaryHandler(tt.Name, tt.Func)
	}
}

func TestWrapOnewayHandlerInvalid(t *testing.T) {
	tests := []struct {
		Name string
		Func interface{}
	}{
		{"empty", func() {}},
		{"not-a-function", 0},
		{
			"wrong-args-in",
			func(context.Context, yarpc.ReqMeta) error {
				return nil
			},
		},
		{
			"wrong-ctx",
			func(string, yarpc.ReqMeta, *struct{}) error {
				return nil
			},
		},
		{
			"wrong-req-meta",
			func(context.Context, yarpc.CallReqMeta, *struct{}) error {
				return nil
			},
		},
		{
			"wrong-req-body",
			func(context.Context, string, int) error {
				return nil
			},
		},
		{
			"wrong-response",
			func(context.Context, yarpc.ReqMeta, map[string]interface{}) (*struct{}, yarpc.ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"wrong-response-val",
			func(context.Context, yarpc.ReqMeta, map[string]interface{}) int {
				return 0
			},
		},
		{
			"non-pointer-req",
			func(context.Context, yarpc.ReqMeta, struct{}) error {
				return nil
			},
		},
		{
			"non-string-key",
			func(context.Context, yarpc.ReqMeta, map[int32]interface{}) error {
				return nil
			},
		},
	}

	for _, tt := range tests {
		assert.Panics(t, assert.PanicTestFunc(func() {
			wrapOnewayHandler(tt.Name, tt.Func)
		}))
	}
}
func TestWrapOnewayHandlerValid(t *testing.T) {
	tests := []struct {
		Name string
		Func interface{}
	}{
		{
			"foo",
			func(context.Context, yarpc.ReqMeta, *struct{}) error {
				return nil
			},
		},
		{
			"bar",
			func(context.Context, yarpc.ReqMeta, map[string]interface{}) error {
				return nil
			},
		},
		{
			"baz",
			func(context.Context, yarpc.ReqMeta, map[string]interface{}) error {
				return nil
			},
		},
		{
			"qux",
			func(context.Context, yarpc.ReqMeta, interface{}) error {
				return nil
			},
		},
	}

	for _, tt := range tests {
		wrapOnewayHandler(tt.Name, tt.Func)
	}
}
