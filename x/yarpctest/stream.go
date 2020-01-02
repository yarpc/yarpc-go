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

package yarpctest

import (
	"bytes"
	"io"
	"io/ioutil"

	"go.uber.org/yarpc/x/yarpctest/types"
)

// SendStreamMsg sends a message to a stream.
func SendStreamMsg(sendMsg string) *types.SendStreamMsg {
	return &types.SendStreamMsg{
		BodyFunc: func() io.ReadCloser {
			return ioutil.NopCloser(bytes.NewBufferString(sendMsg))
		},
	}
}

// SendStreamMsgAndExpectError sends a message on a stream and asserts on the
// error returned.
func SendStreamMsgAndExpectError(sendMsg string, wantErrMsgs ...string) *types.SendStreamMsg {
	return &types.SendStreamMsg{
		BodyFunc: func() io.ReadCloser {
			return ioutil.NopCloser(bytes.NewBufferString(sendMsg))
		},
		WantErrMsgs: wantErrMsgs,
	}
}

// SendStreamDecodeErrorAndExpectError induces a decode error on the stream
// message and asserts on the result.
func SendStreamDecodeErrorAndExpectError(decodeErr error, wantErrMsgs ...string) *types.SendStreamMsg {
	return &types.SendStreamMsg{
		BodyFunc: func() io.ReadCloser {
			return ioutil.NopCloser(readErr{decodeErr})
		},
		WantErrMsgs: wantErrMsgs,
	}
}

type readErr struct {
	err error
}

func (r readErr) Read(p []byte) (n int, err error) {
	return 0, r.err
}

// RecvStreamMsg waits to receive a message on a client stream.
func RecvStreamMsg(wantMsg string) *types.RecvStreamMsg {
	return &types.RecvStreamMsg{WantBody: bytes.NewBufferString(wantMsg).Bytes()}
}

// RecvStreamErr waits to receive a message on a client stream.  It expects
// an error.
func RecvStreamErr(wantErrMsgs ...string) *types.RecvStreamMsg {
	return &types.RecvStreamMsg{WantErrMsgs: wantErrMsgs}
}
