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

package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/yarpc/internal/testutils"
)

const (
	testInput = `get foo
get foo
set foo bar
get foo
get foo
set baz qux
get baz
get foo
get baz
fire foo
fire bar
fired-values
exit`
	testOutput = `get foo
get foo failed: foo
get foo
get foo failed: foo
set foo bar
get foo
foo = bar
get foo
foo = bar
set baz qux
get baz
baz = qux
get foo
foo = bar
get baz
baz = qux
fire foo
fire bar
fired-values
foo bar
exit`
)

func TestExample(t *testing.T) {
	t.Parallel()
	for _, transportType := range testutils.AllTransportTypes {
		t.Run(transportType.String(), func(t *testing.T) {
			testExample(t, transportType, false)
		})
	}
	t.Run("google-grpc", func(t *testing.T) {
		testExample(t, testutils.TransportTypeGRPC, true)
	})
}

func testExample(t *testing.T, transportType testutils.TransportType, googleGRPC bool) {
	output := bytes.NewBuffer(nil)
	require.NoError(t, run(transportType, googleGRPC, false, strings.NewReader(testInput), output))
	require.Equal(t, testOutput, strings.TrimSpace(output.String()))
}
