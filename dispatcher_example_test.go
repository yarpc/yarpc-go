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

package yarpc_test

import (
	"context"
	"fmt"
	"log"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/json"
	"go.uber.org/yarpc/encoding/raw"
)

func ExampleDispatcher_minimal() {
	dispatcher := yarpc.NewDispatcher(yarpc.Config{Name: "myFancyService"})
	if err := dispatcher.Start(); err != nil {
		log.Fatal(err)
	}
	defer dispatcher.Stop()
}

// global dispatcher used in the registration examples
var dispatcher = yarpc.NewDispatcher(yarpc.Config{Name: "service"})

func ExampleDispatcher_Register_raw() {
	handler := func(ctx context.Context, data []byte) ([]byte, error) {
		return data, nil
	}

	dispatcher.Register(raw.Procedure("echo", handler))
}

// Excuse the weird naming of this function. This lets is show as "JSON"
// rather than "Json"

func ExampleDispatcher_Register_jSON() {
	handler := func(ctx context.Context, key string) (string, error) {
		fmt.Println("key", key)
		return "value", nil
	}

	dispatcher.Register(json.Procedure("get", handler))
}
