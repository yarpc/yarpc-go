// Copyright (c) 2024 Uber Technologies, Inc.
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

package yarpcerrors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodesMarshalText(t *testing.T) {
	for code := range _codeToString {
		t.Run(code.String(), func(t *testing.T) {
			text, err := code.MarshalText()
			require.NoError(t, err)
			var unmarshalledCode Code
			require.NoError(t, unmarshalledCode.UnmarshalText(text))
			require.Equal(t, code, unmarshalledCode)
		})
	}
}

func TestCodesMarshalJSON(t *testing.T) {
	for code := range _codeToString {
		t.Run(code.String(), func(t *testing.T) {
			text, err := code.MarshalJSON()
			require.NoError(t, err)
			var unmarshalledCode Code
			require.NoError(t, unmarshalledCode.UnmarshalJSON(text))
			require.Equal(t, code, unmarshalledCode)
		})
	}
}

func TestCodesMapOneToOneAndCovered(t *testing.T) {
	require.Equal(t, len(_codeToString), len(_stringToCode))
	for code, s := range _codeToString {
		otherCode, ok := _stringToCode[s]
		require.True(t, ok)
		require.Equal(t, code, otherCode)
	}
}

func TestCodesFailures(t *testing.T) {
	badCode := Code(100)
	assert.Equal(t, "100", badCode.String())
	_, err := badCode.MarshalText()
	assert.Error(t, err)
	_, err = badCode.MarshalJSON()
	assert.Error(t, err)
	assert.Error(t, badCode.UnmarshalText([]byte("200")))
	assert.Error(t, badCode.UnmarshalJSON([]byte("200")))
	assert.Error(t, badCode.UnmarshalJSON([]byte(`"200"`)))
}
