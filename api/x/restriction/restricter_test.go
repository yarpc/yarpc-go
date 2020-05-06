// Copyright (c) 2020 Uber Technologies, Inc.
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

package restriction

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/api/transport"
)

func TestRestricter(t *testing.T) {
	const (
		testEncoding  = transport.Encoding("test-encoding")
		testTransport = "test-transport"
	)

	t.Run("no tuples", func(t *testing.T) {
		c, err := NewChecker( /* empty */ )
		assert.Error(t, err, "expected successful creation")
		assert.Nil(t, c, "expected nil checker")
	})

	t.Run("whitelisted tuple", func(t *testing.T) {
		r, err := NewChecker(Tuple{
			Transport: testTransport,
			Encoding:  testEncoding,
		})
		require.NoError(t, err, "expected successful creation")

		t.Run("success", func(t *testing.T) {
			err = r.Check(testEncoding, testTransport)
			assert.NoError(t, err, "expected success")
		})

		t.Run("err", func(t *testing.T) {
			err = r.Check("enc", "trans")
			assert.EqualError(t, err,
				`"trans/enc" is not a whitelisted combination, available: "test-transport/test-encoding"`)
		})
	})

	t.Run("invalid tuple", func(t *testing.T) {
		_, err := NewChecker(Tuple{
			Transport: "",
			Encoding:  "",
		})
		require.EqualError(t, err, "tuple missing must have all fields set")
	})
}
