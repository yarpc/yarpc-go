// Copyright (c) 2021 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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
		{
			"second-return-value-not-error",
			func(context.Context, *struct{}) (*struct{}, *struct{}) {
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
