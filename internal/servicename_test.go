// Copyright (c) 2019 Uber Technologies, Inc.
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

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidServiceNames(t *testing.T) {
	tests := []string{
		"aa",
		"superskipper",
		"supper-skipper",
		"supperskipper83",
		"supper-skipper83",
		"a77ab7g4-51cb-4808-a9ef-875568bde54a", // not valid UUID
		"eviluuid26695g10-a384-48e7-8867-6d48b7fae80a", // not valid UUID
		"a77gb7e4-51cb-4808-a9ef-875568bde54aeviluuid", // not valid UUID
	}
	for _, n := range tests {
		assert.NoError(t, ValidateServiceName(n), "Expected %q to be a valid service name.", n)
	}
}

func TestInvalidServiceNames(t *testing.T) {
	tests := []string{
		"a",
		"a77ab7e4-51cb-4808-a9ef-875568bde54a",
		"urn:uuid:a77ab7e4-51cb-4808-a9ef-875568bde54a",
		"26695a10-a384-48e7-8867-6d48b7fae80a",
		"eviluuid26695a10-a384-48e7-8867-6d48b7fae80a",
		"a77ab7e4-51cb-4808-a9ef-875568bde54aeviluuid",
		"26695g10-a384-48e7-8867-6d48b7fae80a", // not valid UUID, but starts with a number.
		"superSkipper",
		"SuperSkipper",
		"super_skipper",
		"083superskipper",
		"茶",
		"super skipper",
		"100",
		"10-09-2016",
		"",
		"    ",
		"-",
		"no--duplication",
		"no---duplication",
		"endswithadash-",
		"endswithasterisk*",
		"internal*-asterisk",
	}
	for _, n := range tests {
		assert.Error(t, ValidateServiceName(n), "Expected %q to be an invalid service name", n)
	}
}
