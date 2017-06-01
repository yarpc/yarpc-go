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

package ioutil

import (
	"bytes"
	"io"
	"testing"
)

func TestRereader(t *testing.T) {
	type testStruct struct {
		msg string

		// Source reader that will be used for all reads
		source io.Reader

		// A list of actions that will be applied on the Rereader
		actions []RereaderAction
	}
	tests := []testStruct{
		{
			msg:    "read once",
			source: bytes.NewBufferString("this is a test"),
			actions: []RereaderAction{
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
			},
		},
		{
			msg:    "read twice",
			source: bytes.NewBufferString("this is a test"),
			actions: []RereaderAction{
				ReadAction{
					Into:      make([]byte, 4),
					WantBytes: []byte("this"),
					WantN:     4,
				},
				ReadAction{
					Into:      make([]byte, 10),
					WantBytes: []byte(" is a test"),
					WantN:     10,
				},
			},
		},
		{
			msg:    "read into larger buffer",
			source: bytes.NewBufferString("this is a test"),
			actions: []RereaderAction{
				ReadAction{
					Into:      []byte("-------------------"),
					WantBytes: []byte("this is a test-----"),
					WantN:     14,
				},
			},
		},
		{
			msg:    "read end of file error",
			source: bytes.NewBufferString("this is a test"),
			actions: []RereaderAction{
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
				ReadAction{
					Into:      make([]byte, 10),
					WantBytes: make([]byte, 10),
					WantError: io.EOF,
				},
			},
		},
		{
			msg:    "reread once",
			source: bytes.NewBufferString("this is a test"),
			actions: []RereaderAction{
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
				ResetAction{},
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
			},
		},
		{
			msg:    "reread partial",
			source: bytes.NewBufferString("this is a test"),
			actions: []RereaderAction{
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
				ResetAction{},
				ReadAction{
					Into:      make([]byte, 4),
					WantBytes: []byte("this"),
					WantN:     4,
				},
				ReadAction{
					Into:      make([]byte, 10),
					WantBytes: []byte(" is a test"),
					WantN:     10,
				},
			},
		},
		{
			msg:    "reread end of file",
			source: bytes.NewBufferString("this is a test"),
			actions: []RereaderAction{
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
				ResetAction{},
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
				ReadAction{
					Into:      make([]byte, 10),
					WantBytes: make([]byte, 10),
					WantError: io.EOF,
				},
			},
		},
		{
			msg:    "reread 5 times",
			source: bytes.NewBufferString("this is a test"),
			actions: []RereaderAction{
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
				ResetAction{},
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
				ResetAction{},
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
				ResetAction{},
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
				ResetAction{},
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
				ResetAction{},
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
			},
		},
		{
			msg:    "reset before read",
			source: bytes.NewBufferString("this is a test"),
			actions: []RereaderAction{
				ResetAction{
					WantError: ResetError,
				},
			},
		},
		{
			msg:    "reset before read finished",
			source: bytes.NewBufferString("this is a test"),
			actions: []RereaderAction{
				ReadAction{
					Into:      make([]byte, 13),
					WantBytes: []byte("this is a tes"),
					WantN:     13,
				},
				ResetAction{
					WantError: ResetError,
				},
			},
		},
		{
			msg:    "reset before second read finished",
			source: bytes.NewBufferString("this is a test"),
			actions: []RereaderAction{
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
				ResetAction{},
				ReadAction{
					Into:      make([]byte, 13),
					WantBytes: []byte("this is a tes"),
					WantN:     13,
				},
				ResetAction{
					WantError: ResetError,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			reader, closer := NewRereader(tt.source)
			defer closer()
			ApplyRereaderActions(t, reader, tt.actions)
		})
	}
}
