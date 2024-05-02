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

package procedure

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcedureSplitEmpty(t *testing.T) {
	s, m := FromName("")
	assert.Equal(t, "", s)
	assert.Equal(t, "", m)
}

func TestProcedureNameAndSplit(t *testing.T) {
	tests := []struct {
		Procedure string
		Service   string
		Method    string
	}{
		{"foo::bar", "foo", "bar"},
		{"::bar", "", "bar"},
		{"foo::", "foo", ""},
		{"::", "", ""},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.Procedure, ToName(tt.Service, tt.Method))
		s, m := FromName(tt.Procedure)
		assert.Equal(t, tt.Service, s)
		assert.Equal(t, tt.Method, m)
	}
}

func BenchmarkToName(b *testing.B) {
	benchmarks := []struct {
		desc         string
		stringLength int
	}{
		{"StringsLength-10", 10},
		{"StringsLength-100", 100},
		{"StringsLength-1000", 1000},
	}

	for _, benchmark := range benchmarks {
		b.Run(benchmark.desc, func(b *testing.B) {
			serviceName := randomString(benchmark.stringLength)
			methodName := randomString(benchmark.stringLength)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ToName(serviceName, methodName)
			}
		})
	}
}

var _letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func randomString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = _letterRunes[rand.Intn(len(_letterRunes))]
	}
	return string(b)
}
