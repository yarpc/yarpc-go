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
			source: newWrappedReader("this is a test"),
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
			source: newWrappedReader("this is a test"),
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
			source: newWrappedReader("this is a test"),
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
			source: newWrappedReader("this is a test"),
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
			source: newWrappedReader("this is a test"),
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
			source: newWrappedReader("this is a test"),
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
			source: newWrappedReader("this is a test"),
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
			source: newWrappedReader("this is a test"),
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
			source: newWrappedReader("this is a test"),
			actions: []RereaderAction{
				ResetAction{},
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
			},
		},
		{
			msg:    "reset after partial read",
			source: newWrappedReader("this is a test"),
			actions: []RereaderAction{
				ReadAction{
					Into:      make([]byte, 13),
					WantBytes: []byte("this is a tes"),
					WantN:     13,
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
			msg:    "reset after second partial read",
			source: newWrappedReader("this is a test"),
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
				ResetAction{},
				ReadAction{
					Into:      make([]byte, 14),
					WantBytes: []byte("this is a test"),
					WantN:     14,
				},
			},
		},
		{
			msg:    "use buffer immediately",
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
			msg:    "reuse rereader",
			source: newRereaderWithoutClosure(newWrappedReader("this is a test")),
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

type wrappedReader struct {
	buf *bytes.Buffer
}

func (w *wrappedReader) Read(p []byte) (n int, err error) {
	return w.buf.Read(p)
}

func newWrappedReader(str string) io.Reader {
	return &wrappedReader{
		buf: bytes.NewBufferString(str),
	}
}

func newRereaderWithoutClosure(src io.Reader) *Rereader {
	rr, _ := NewRereader(src)
	return rr
}
