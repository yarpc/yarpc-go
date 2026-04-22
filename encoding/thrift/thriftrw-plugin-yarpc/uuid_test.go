// Copyright (c) 2026 Uber Technologies, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/thriftrw/compile"
	"go.uber.org/thriftrw/ptr"
	noservices "go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc/internal/tests/NOSERVICES"
)

func TestGetUUID(t *testing.T) {
	t.Run("happyCase", func(t *testing.T) {
		assert.NotPanics(t, func() {
			spec, err := compile.Compile("internal/uuid_test_thrift/happy.thrift")
			assert.NoError(t, err)
			annotatedTypes, err := anyAnnotatedTypes(spec.Types)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(annotatedTypes))
			for k, v := range annotatedTypes {
				assert.Equal(t, "Struct", k.Name)
				assert.Equal(t, v, "UserIdentifier")
			}

		})
	})
	t.Run("multipleAnnotatedStructs", func(t *testing.T) {
		assert.NotPanics(t, func() {
			spec, err := compile.Compile("internal/uuid_test_thrift/multipleAnnotatedStructs.thrift")
			assert.NoError(t, err)
			annotatedTypes, err := anyAnnotatedTypes(spec.Types)
			assert.NoError(t, err)
			assert.Equal(t, 2, len(annotatedTypes))
			uuidFields := map[string]string{
				"RedStruct":   "UserIdentifier",
				"GreenStruct": "CatIdentifier",
			}
			for k, v := range annotatedTypes {
				assert.Equal(t, uuidFields[k.Name], v)
			}

		})
	})

	t.Run("errorCase", func(t *testing.T) {
		assert.NotPanics(t, func() {
			spec, err := compile.Compile("internal/uuid_test_thrift/broken.thrift")
			assert.NoError(t, err)
			_, err = anyAnnotatedTypes(spec.Types)
			assert.Error(t, err)

		})
	})
}

func TestGetGeneratedUUID(t *testing.T) {
	t.Run("Found the UUID", func(t *testing.T) {
		st := noservices.Struct{
			Baz:            ptr.String("asdf"),
			UserIdentifier: ptr.String("my-uuid"),
		}
		assert.Equal(t, "my-uuid", st.ActorUUID())
	})
}
