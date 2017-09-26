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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFeatureIn(t *testing.T) {
	feature1 := Feature(200)
	feature2 := Feature(201)
	require.False(t, feature1.In(nil))
	require.False(t, feature1.In([]Feature{}))
	require.False(t, feature1.In([]Feature{feature2}))
	require.True(t, feature1.In([]Feature{feature1}))
	require.True(t, feature1.In([]Feature{feature2, feature1}))
	require.True(t, feature1.In([]Feature{feature1, feature2}))
}

func TestFeaturesFromString(t *testing.T) {
	for feature := range _featureToString {
		t.Run(feature.String(), func(t *testing.T) {
			gotFeature, ok := FeatureFromString(feature.String())
			require.True(t, ok)
			require.Equal(t, feature, gotFeature)
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

func TestFeaturesLowercaseAndNoCommas(t *testing.T) {
	for feature := range _featureToString {
		t.Run(feature.String(), func(t *testing.T) {
			s := feature.String()
			require.Equal(t, s, strings.ToLower(s))
			require.False(t, strings.Contains(s, ","))
		})
	}
}
