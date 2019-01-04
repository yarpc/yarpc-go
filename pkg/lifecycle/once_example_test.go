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

package lifecycle_test

import (
	"fmt"

	"go.uber.org/yarpc/pkg/lifecycle"
)

// Engine is an example of a type that uses a lifecycle.Once to synchronize its
// lifecycle.
type Engine struct {
	once *lifecycle.Once
}

// NewEngine returns a lifecycle example.
func NewEngine() (*Engine, error) {
	return &Engine{
		once: lifecycle.NewOnce(),
	}, nil
}

// Start advances the engine to the running state (if it has not already done
// so), printing "started".
func (e *Engine) Start() error {
	return e.once.Start(e.start)
}

func (e *Engine) start() error {
	fmt.Printf("started\n")
	return nil
}

// Stop advances the engine to the stopped state (if it has not already done
// so), printing "stopped".
func (e *Engine) Stop() error {
	return e.once.Stop(e.stop)
}

func (e *Engine) stop() error {
	fmt.Printf("stopped\n")
	return nil
}

func Example() {
	engine, err := NewEngine()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	go engine.Start() // might win race to start
	engine.Start()    // blocks until started
	defer engine.Stop()

	// Output:
	// started
	// stopped
}
