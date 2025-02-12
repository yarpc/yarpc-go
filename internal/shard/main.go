// Copyright (c) 2025 Uber Technologies, Inc.
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

package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

// Shards a list of files over a distribution defined by totalShards.
// Useful for splitting work and running tests in parallel.
var usage = fmt.Sprintf("Usage: %s shardNum totalShards args...", os.Args[0])

func main() {
	if err := do(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func do(args []string, writer io.Writer) error {
	if len(args) < 2 {
		return errors.New(usage)
	}
	shardNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("%v\n%s", err, usage)
	}
	totalShards, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("%v\n%s", err, usage)
	}
	for i := 2; i < len(args); i++ {
		if (((i - 2) % totalShards) + 1) == shardNum {
			if i != 2 {
				fmt.Fprint(writer, " ")
			}
			fmt.Fprint(writer, args[i])
		}
	}
	return nil
}
