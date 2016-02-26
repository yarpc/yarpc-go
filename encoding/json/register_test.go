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
			func(*Request, map[string]interface{}) (*Response, error) {
				return nil, nil
			},
		},
		{
			"wrong-request",
			func(string, *struct{}) (*struct{}, *Response, error) {
				return nil, nil, nil
			},
		},
		{
			"non-pointer-req",
			func(*Request, struct{}) (*struct{}, *Response, error) {
				return nil, nil, nil
			},
		},
		{
			"non-pointer-res",
			func(*Request, *struct{}) (struct{}, *Response, error) {
				return struct{}{}, nil, nil
			},
		},
		{
			"non-string-key",
			func(*Request, map[int32]interface{}) (*struct{}, *Response, error) {
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
			func(*Request, *struct{}) (*struct{}, *Response, error) {
				return nil, nil, nil
			},
		},
		{
			"bar",
			func(*Request, map[string]interface{}) (*struct{}, *Response, error) {
				return nil, nil, nil
			},
		},
		{
			"baz",
			func(*Request, map[string]interface{}) (map[string]interface{}, *Response, error) {
				return nil, nil, nil
			},
		},
	}

	for _, tt := range tests {
		wrapHandler(tt.Name, tt.Func)
	}
}
