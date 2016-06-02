package json

import (
	"testing"

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
			func(*ReqMeta, map[string]interface{}) (*ResMeta, error) {
				return nil, nil
			},
		},
		{
			"wrong-request",
			func(string, *struct{}) (*struct{}, *ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"non-pointer-req",
			func(*ReqMeta, struct{}) (*struct{}, *ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"non-pointer-res",
			func(*ReqMeta, *struct{}) (struct{}, *ResMeta, error) {
				return struct{}{}, nil, nil
			},
		},
		{
			"non-string-key",
			func(*ReqMeta, map[int32]interface{}) (*struct{}, *ResMeta, error) {
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
			func(*ReqMeta, *struct{}) (*struct{}, *ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"bar",
			func(*ReqMeta, map[string]interface{}) (*struct{}, *ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"baz",
			func(*ReqMeta, map[string]interface{}) (map[string]interface{}, *ResMeta, error) {
				return nil, nil, nil
			},
		},
		{
			"qux",
			func(*ReqMeta, interface{}) (map[string]interface{}, *ResMeta, error) {
				return nil, nil, nil
			},
		},
	}

	for _, tt := range tests {
		wrapHandler(tt.Name, tt.Func)
	}
}
