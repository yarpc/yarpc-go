// Copyright (c) 2017 Uber Technologies, Inc.
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

package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeaturesMarshalText(t *testing.T) {
	for feature := range _featureToString {
		t.Run(feature.String(), func(t *testing.T) {
			text, err := feature.MarshalText()
			require.NoError(t, err)
			var unmarshalledFeature Feature
			require.NoError(t, unmarshalledFeature.UnmarshalText(text))
			require.Equal(t, feature, unmarshalledFeature)
		})
	}
}

func TestFeaturesMarshalJSON(t *testing.T) {
	for feature := range _featureToString {
		t.Run(feature.String(), func(t *testing.T) {
			text, err := feature.MarshalJSON()
			require.NoError(t, err)
			var unmarshalledFeature Feature
			require.NoError(t, unmarshalledFeature.UnmarshalJSON(text))
			require.Equal(t, feature, unmarshalledFeature)
		})
	}
}

func TestFeaturesMapOneToOneAndCovered(t *testing.T) {
	require.Equal(t, len(_featureToString), len(_stringToFeature))
	for feature, s := range _featureToString {
		otherFeature, ok := _stringToFeature[s]
		require.True(t, ok)
		require.Equal(t, feature, otherFeature)
	}
}

func TestFeaturesFailures(t *testing.T) {
	badFeature := Feature(100)
	assert.Equal(t, "100", badFeature.String())
	_, err := badFeature.MarshalText()
	assert.Error(t, err)
	_, err = badFeature.MarshalJSON()
	assert.Error(t, err)
	assert.Error(t, badFeature.UnmarshalText([]byte("200")))
	assert.Error(t, badFeature.UnmarshalJSON([]byte("200")))
	assert.Error(t, badFeature.UnmarshalJSON([]byte(`"200"`)))
}
