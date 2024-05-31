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

package tchclient

import (
	"fmt"

	"github.com/crossdock/crossdock-go"
	"github.com/uber/tchannel-go"
	"go.uber.org/yarpc/internal/crossdock/client/params"
)

const (
	serverPort = 8082
	serverName = "yarpc-test"
)

// Run exercises a YARPC server from a tchannel client.
func Run(t crossdock.T) {
	fatals := crossdock.Fatals(t)

	encoding := t.Param(params.Encoding)
	server := t.Param(params.Server)
	serverHostPort := fmt.Sprintf("%v:%v", server, serverPort)

	ch, err := tchannel.NewChannel("tchannel-client", nil)
	fatals.NoError(err, "Could not create channel")

	call := call{Channel: ch, ServerHostPort: serverHostPort}

	switch encoding {
	case "raw":
		runRaw(t, call)
	case "json":
		runJSON(t, call)
	case "thrift":
		runThrift(t, call)
	default:
		fatals.Fail("", "unknown encoding %q", encoding)
	}
}

// call contains the details needed for each tchannel scheme
// to make an outbound call. Because the way you connect is not uniform
// between schemes, it's not enough to just add a peer and go.
type call struct {
	Channel        *tchannel.Channel
	ServerHostPort string
}
