package json

import (
	"context"
	"testing"

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
			func(context.Context) (*struct{}, error) {
				return nil, nil
			},
		},
		{
			"wrong-ctx",
			func(string, *struct{}) (*struct{}, error) {
				return nil, nil
			},
		},
		{
			"wrong-req-body",
			func(context.Context, string, int) (*struct{}, error) {
				return nil, nil
			},
		},
		{
			"wrong-response",
			func(context.Context, map[string]interface{}) error {
				return nil
			},
		},
		{
			"non-pointer-req",
			func(context.Context, struct{}) (*struct{}, error) {
				return nil, nil
			},
		},
		{
			"non-pointer-res",
			func(context.Context, *struct{}) (struct{}, error) {
				return struct{}{}, nil
			},
		},
		{
			"non-string-key",
			func(context.Context, map[int32]interface{}) (*struct{}, error) {
				return nil, nil
			},
		},
	}

	for _, tt := range tests {
		assert.Panics(t, assert.PanicTestFunc(func() {
			wrapUnaryHandler(tt.Name, tt.Func)
		}), tt.Name)
	}
}

func TestWrapUnaryHandlerValid(t *testing.T) {
	tests := []struct {
		Name string
		Func interface{}
	}{
		{
			"foo",
			func(context.Context, *struct{}) (*struct{}, error) {
				return nil, nil
			},
		},
		{
			"bar",
			func(context.Context, map[string]interface{}) (*struct{}, error) {
				return nil, nil
			},
		},
		{
			"baz",
			func(context.Context, map[string]interface{}) (map[string]interface{}, error) {
				return nil, nil
			},
		},
		{
			"qux",
			func(context.Context, interface{}) (map[string]interface{}, error) {
				return nil, nil
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
			func(context.Context) error {
				return nil
			},
		},
		{
			"wrong-ctx",
			func(string, *struct{}) error {
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
			func(context.Context, map[string]interface{}) (*struct{}, error) {
				return nil, nil
			},
		},
		{
			"wrong-response-val",
			func(context.Context, map[string]interface{}) int {
				return 0
			},
		},
		{
			"non-pointer-req",
			func(context.Context, struct{}) error {
				return nil
			},
		},
		{
			"non-string-key",
			func(context.Context, map[int32]interface{}) error {
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
			func(context.Context, *struct{}) error {
				return nil
			},
		},
		{
			"bar",
			func(context.Context, map[string]interface{}) error {
				return nil
			},
		},
		{
			"baz",
			func(context.Context, map[string]interface{}) error {
				return nil
			},
		},
		{
			"qux",
			func(context.Context, interface{}) error {
				return nil
			},
		},
	}

	for _, tt := range tests {
		wrapOnewayHandler(tt.Name, tt.Func)
	}
}
