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

package stream

import (
	"fmt"
	"io"
	"strings"

	streampb "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go2/internal/tests/gen/proto/src/stream"
)

type helloServer struct{}

// NewServer returns a new streampb.StreamerServer.
func NewServer() streampb.HelloYARPCServer {
	return &helloServer{}
}

// In represents a simple server-side streaming procedure. The number of messages
// the server responds with is equal to the length of the request's greeting.
// If the client sends a greeting "hello", for example, the server will respond
// with five individual messages.
func (h *helloServer) In(req *streampb.HelloRequest, s streampb.HelloInYARPCStreamServer) error {
	for i := 0; i < len(req.GetGreeting()); i++ {
		resp := fmt.Sprintf("Received %d", i)
		if err := s.Send(&streampb.HelloResponse{Response: resp}); err != nil {
			return err
		}
	}
	return nil
}

// Out represents a simple client-side streaming procedure. The server will
// collect greetings received from the stream and join them together in its
// response.
func (h *helloServer) Out(s streampb.HelloOutYARPCStreamServer) (*streampb.HelloResponse, error) {
	var msgs []string
	for {
		req, err := s.Recv()
		if err == io.EOF {
			resp := fmt.Sprintf("Received %v", strings.Join(msgs, ","))
			return &streampb.HelloResponse{Response: resp}, nil
		}
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, req.GetGreeting())
	}
}

// Bidirectional represents a simple bidirectional streaming procedure. The server
// will continue to respond to greetings until it receives "exit".
func (h *helloServer) Bidirectional(s streampb.HelloBidirectionalYARPCStreamServer) error {
	for {
		req, err := s.Recv()
		if err != nil {
			return err
		}

		if req.GetGreeting() == "exit" {
			return nil
		}

		resp := fmt.Sprintf("Received %q", req.GetGreeting())
		err = s.Send(&streampb.HelloResponse{Response: resp})
		if err != nil {
			return err
		}
	}
}
