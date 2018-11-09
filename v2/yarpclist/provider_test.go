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

package yarpclist

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/v2/yarpctest"
)

func TestProvider(t *testing.T) {
	t.Run("registration conflict", func(t *testing.T) {
		_, err := NewProvider(
			yarpctest.NewFakePeerList("foo"),
			yarpctest.NewFakePeerList("foo"),
		)
		assert.EqualError(t, err, `list "foo" was registered more than once`)
	})

	t.Run("successful construction", func(t *testing.T) {
		p, err := NewProvider(yarpctest.NewFakePeerList("foo"))
		require.NoError(t, err)

		list, ok := p.List("foo")
		require.True(t, ok)
		assert.Equal(t, "foo", list.Name())
	})
}
