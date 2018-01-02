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

package pallytest

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Scrape collects and returns the plain-text content of the registry's scrape
// endpoint, along with the response code.
func Scrape(t testing.TB, registry http.Handler) (int, string) {
	server := httptest.NewServer(registry)
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err, "Unexpected error scraping Prometheus endpoint.")
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err, "Unexpected error reading response body.")
	return resp.StatusCode, strings.TrimSpace(string(body))
}

// AssertPrometheus asserts that the registry's scrape endpoint successfully
// serves the supplied plain-text Prometheus metrics.
func AssertPrometheus(t testing.TB, registry http.Handler, expected string) {
	code, actual := Scrape(t, registry)
	assert.Equal(t, http.StatusOK, code, "Unexpected HTTP response code from Prometheus scrape.")
	assert.Equal(
		t,
		strings.Split(expected, "\n"),
		strings.Split(actual, "\n"),
		"Unexpected Prometheus text.",
	)
}
