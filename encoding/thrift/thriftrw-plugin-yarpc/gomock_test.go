// Copyright (c) 2018 Uber Technologies, Inc.
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

package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/thriftrw/ptr"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/readonlystoretest"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/atomic/storetest"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/baseservicetest"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/emptyservicetest"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/extendemptytest"
	"go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/common/extendonlytest"
)

func TestMockClients(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		desc           string
		withController func(*gomock.Controller)

		wantStatus     FakeTestStatus
		wantPanic      interface{}
		wantErrorsLike []string
	}{
		{
			desc: "empty",
			withController: func(ctrl *gomock.Controller) {
				emptyservicetest.NewMockClient(ctrl).EXPECT()
			},
		},
		{
			desc: "extends empty: unexpected",
			withController: func(ctrl *gomock.Controller) {
				c := extendemptytest.NewMockClient(ctrl)
				c.Hello(ctx)
			},
			wantStatus:     Fatal,
			wantErrorsLike: []string{"no matching expected call"},
		},
		{
			desc: "extends empty: expected:",
			withController: func(ctrl *gomock.Controller) {
				c := extendemptytest.NewMockClient(ctrl)
				c.EXPECT().Hello(gomock.Any()).Return(nil)
				assert.NoError(t, c.Hello(ctx))
			},
		},
		{
			desc: "extends empty: missing",
			withController: func(ctrl *gomock.Controller) {
				c := extendemptytest.NewMockClient(ctrl)
				c.EXPECT().Hello(gomock.Any()).Return(nil)
			},
			wantStatus: Fatal,
			wantErrorsLike: []string{
				"missing call(s) to [*extendemptytest.MockClient.Hello(is anything)]",
				"aborting test due to missing call(s)",
			},
		},
		{
			desc: "extends only: unexpected",
			withController: func(ctrl *gomock.Controller) {
				c := extendonlytest.NewMockClient(ctrl)
				c.Healthy(ctx)
			},
			wantStatus:     Fatal,
			wantErrorsLike: []string{"no matching expected call"},
		},
		{
			desc: "extends only: expected:",
			withController: func(ctrl *gomock.Controller) {
				c := extendonlytest.NewMockClient(ctrl)
				c.EXPECT().Healthy(gomock.Any()).Return(true, nil)
				healthy, err := c.Healthy(ctx)
				assert.NoError(t, err)
				assert.True(t, healthy)
			},
		},
		{
			desc: "extends only: missing",
			withController: func(ctrl *gomock.Controller) {
				c := extendonlytest.NewMockClient(ctrl)
				c.EXPECT().Healthy(gomock.Any()).Return(false, nil)
			},
			wantStatus: Fatal,
			wantErrorsLike: []string{
				"missing call(s) to [*extendonlytest.MockClient.Healthy(is anything)]",
				"aborting test due to missing call(s)",
			},
		},
		{
			desc: "base: expected with options",
			withController: func(ctrl *gomock.Controller) {
				c := baseservicetest.NewMockClient(ctrl)
				c.EXPECT().Healthy(gomock.Any(), gomock.Any()).Return(true, nil)

				// NOTE: Each option has to have a different argument in the
				// EXPECT call. We should figure out an API to add assertions
				// on options. See https://github.com/yarpc/yarpc-go/issues/683

				healthy, err := c.Healthy(ctx, yarpc.WithHeader("key", "value"))
				assert.True(t, healthy)
				assert.NoError(t, err)
			},
		},
		{
			desc: "store: healthy: unexpected call",
			withController: func(ctrl *gomock.Controller) {
				c := storetest.NewMockClient(ctrl)
				c.Healthy(ctx)
			},
			wantStatus:     Fatal,
			wantErrorsLike: []string{"no matching expected call"},
		},
		{
			desc: "store: healthy: expected call",
			withController: func(ctrl *gomock.Controller) {
				c := storetest.NewMockClient(ctrl)
				c.EXPECT().Healthy(gomock.Any()).Return(true, nil)

				result, err := c.Healthy(ctx)
				assert.NoError(t, err, "mock should return no error")
				assert.True(t, result, "mock should return true")
			},
		},
		{
			desc: "store: healthy: expected call: throw error",
			withController: func(ctrl *gomock.Controller) {
				c := storetest.NewMockClient(ctrl)
				c.EXPECT().Healthy(gomock.Any()).Return(false, errors.New("great sadness"))

				_, err := c.Healthy(ctx)
				assert.Equal(t, errors.New("great sadness"), err)
			},
		},
		{
			desc: "readonly store: integer: missing",
			withController: func(ctrl *gomock.Controller) {
				c := readonlystoretest.NewMockClient(ctrl)
				c.EXPECT().Integer(gomock.Any(), ptr.String("foo")).Return(int64(42), nil)
			},
			wantStatus: Fatal,
			wantErrorsLike: []string{
				"missing call(s) to [*readonlystoretest.MockClient.Integer(is anything",
				"aborting test due to missing call(s)",
			},
		},
		{
			desc: "store: integer: expected",
			withController: func(ctrl *gomock.Controller) {
				c := storetest.NewMockClient(ctrl)
				c.EXPECT().Integer(gomock.Any(), ptr.String("foo")).Return(int64(42), nil)
				result, err := c.Integer(ctx, ptr.String("foo"))
				assert.NoError(t, err)
				assert.Equal(t, int64(42), result)
			},
		},
		{
			desc: "store: forget: unexpected",
			withController: func(ctrl *gomock.Controller) {
				c := storetest.NewMockClient(ctrl)
				c.Forget(ctx, ptr.String("hello"))
			},
			wantStatus:     Fatal,
			wantErrorsLike: []string{"no matching expected call"},
		},
		{
			desc: "store: forget: expected",
			withController: func(ctrl *gomock.Controller) {
				c := storetest.NewMockClient(ctrl)
				c.EXPECT().Forget(gomock.Any(), ptr.String("hello")).Return(nil, errors.New("great sadness"))

				_, err := c.Forget(ctx, ptr.String("hello"))
				assert.Equal(t, errors.New("great sadness"), err)
			},
		},
		{
			desc: "store: forget: missing",
			withController: func(ctrl *gomock.Controller) {
				c := storetest.NewMockClient(ctrl)
				c.EXPECT().Forget(gomock.Any(), ptr.String("hello")).Return(nil, errors.New("great sadness"))
			},
			wantStatus: Fatal,
			wantErrorsLike: []string{
				"missing call(s) to [*storetest.MockClient.Forget(is anything",
				"aborting test due to missing call(s)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := withFakeTestReporter(func(fakeT gomock.TestReporter) {
				mockCtrl := gomock.NewController(fakeT)
				defer mockCtrl.Finish()

				tt.withController(mockCtrl)
			})

			if !assert.Equal(t, tt.wantStatus, result.Status, "expected %v, got %v", tt.wantStatus, result.Status) {
				if result.Status == Panicked {
					t.Fatalf("panicked: %v\n%v", result.Panic, result.PanicTrace)
				}
			}

			assert.Equal(t, tt.wantPanic, result.Panic)
			assertErrorsMatch(t, tt.wantErrorsLike, result.Errors)
		})
	}
}

func assertErrorsMatch(t *testing.T, wantErrorsLike, errors []string) {
	var unexpectedErrors []string
	for _, err := range errors {
		before := len(wantErrorsLike)
		wantErrorsLike = removeMatchingError(wantErrorsLike, err)
		if len(wantErrorsLike) == before {
			unexpectedErrors = append(unexpectedErrors, err)
			continue
		}
	}

	if len(wantErrorsLike) > 0 {
		msg := "expected but did not receive errors like:"
		for _, m := range wantErrorsLike {
			msg += "\n -  " + indentTail(4, m)
		}
		t.Errorf(msg)
	}

	if len(unexpectedErrors) > 0 {
		msg := "received unexpected errors:"
		for _, err := range unexpectedErrors {
			msg += "\n -  " + indentTail(4, err)
		}
		t.Error(msg)
	}
}

func removeMatchingError(matchers []string, err string) []string {
	match := -1
	for i, m := range matchers {
		if strings.Contains(err, m) {
			match = i
			break
		}
	}

	if match < 0 {
		return matchers
	}

	matchers = append(matchers[:match], matchers[match+1:]...)
	return matchers
}

// indentTail prepends the given number of spaces to all lines following the
// first line of the given string.
func indentTail(spaces int, s string) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i, line := range lines[1:] {
		lines[i+1] = prefix + line
	}
	return strings.Join(lines, "\n")
}
