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

	streampb "go.uber.org/yarpc/v2/yarpcprotobuf/protoc-gen-yarpc-go/internal/tests/gen/proto/src/stream"
)

type helloServer struct{}

// NewServer returns a new streampb.StreamerServer.
func NewServer() streampb.HelloYARPCServer {
	return &helloServer{}
}

func (h *helloServer) In(req *streampb.HelloRequest, s streampb.HelloInYARPCStreamServer) error {
	resp := fmt.Sprintf("Received %q", req.GetGreeting())
	return s.Send(&streampb.HelloResponse{Response: resp})
}

func (h *helloServer) Out(s streampb.HelloOutYARPCStreamServer) (*streampb.HelloResponse, error) {
	req, err := s.Recv()
	if err != nil {
		return nil, err
	}
	resp := fmt.Sprintf("Received %q", req.GetGreeting())
	return &streampb.HelloResponse{Response: resp}, nil
}

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
		if err := s.Send(&streampb.HelloResponse{Response: resp}); err != nil {
			return err
		}
	}
}
