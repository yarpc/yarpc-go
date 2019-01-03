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

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mattn/go-shellwords"
)

type cmd struct {
	cmd           *exec.Cmd
	sleep         time.Duration
	output        string
	debug         bool
	stdout        *bytes.Buffer
	stderr        *bytes.Buffer
	flushedStdout bool
	flushedStderr bool
	finished      bool
	lock          sync.Mutex
}

func newCmd(cmdConfig *cmdConfig, dir string, debug bool) (*cmd, error) {
	parser := shellwords.NewParser()
	parser.ParseEnv = true
	args, err := parser.Parse(cmdConfig.Command)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("command evaulated to empty: %s", cmdConfig.Command)
	}
	execCmd := exec.Command(args[0], args[1:]...)
	execCmd.Dir = dir
	// https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
	execCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd := &cmd{
		execCmd,
		time.Duration(cmdConfig.SleepMs) * time.Millisecond,
		cmdConfig.Output,
		debug,
		bytes.NewBuffer(nil),
		bytes.NewBuffer(nil),
		false,
		false,
		false,
		sync.Mutex{},
	}
	if cmdConfig.Input != "" {
		execCmd.Stdin = strings.NewReader(cmdConfig.Input)
	}
	execCmd.Stdout = cmd.stdout
	execCmd.Stderr = cmd.stderr
	return cmd, nil
}

func (c *cmd) Start() error {
	c.debugPrintf("starting")
	if err := c.cmd.Start(); err != nil {
		return c.wrapError("failed to start", err)
	}
	c.debugPrintf("started")
	if c.sleep != 0 {
		c.debugPrintf("sleeping")
		<-time.After(c.sleep)
		c.debugPrintf("done sleeping")
	}
	return nil
}

func (c *cmd) Wait() error {
	c.debugPrintf("waiting")
	if err := c.cmd.Wait(); err != nil {
		return c.wrapError("failed", err)
	}
	c.debugPrintf("finished")
	c.lock.Lock()
	defer c.lock.Unlock()
	c.finished = true
	return nil
}

func (c *cmd) Validate() error {
	if c.output == "" {
		return nil
	}
	output := cleanOutput(string(c.stdout.Bytes()))
	expectedOutput := cleanOutput(c.output)
	if output != expectedOutput {
		return c.wrapError("validation failed", fmt.Errorf("expected\n%s\ngot\n%s", expectedOutput, output))
	}
	return nil
}

func (c *cmd) Clean(suppressStdout bool) {
	c.Kill()
	if !suppressStdout {
		c.FlushStdout()
	}
	c.FlushStderr()
}

func (c *cmd) Kill() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.finished {
		return
	}
	if c.cmd.Process != nil {
		c.debugPrintf("killing")
		// https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
		_ = syscall.Kill(-c.cmd.Process.Pid, syscall.SIGKILL)
		c.finished = true
	}
}

func (c *cmd) FlushStdout() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.flushedStdout {
		return
	}
	if data := c.stdout.Bytes(); len(data) > 0 {
		fmt.Print(string(data))
	}
	c.flushedStdout = true
}

func (c *cmd) FlushStderr() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.flushedStderr {
		return
	}
	if data := c.stderr.Bytes(); len(data) > 0 {
		fmt.Fprint(os.Stderr, string(data))
	}
	c.flushedStderr = true
}

func (c *cmd) String() string {
	if len(c.cmd.Args) == 0 {
		return c.cmd.Path
	}
	return strings.Join(c.cmd.Args, " ")
}

func (c *cmd) wrapError(msg string, err error) error {
	return fmt.Errorf("%v: %s: %v", c, msg, err)
}

func (c *cmd) debugPrintf(format string, args ...interface{}) {
	if c.debug {
		args = append([]interface{}{c}, args...)
		log.Printf("%v: "+format, args...)
	}
}

func cleanOutput(output string) string {
	output = strings.TrimSpace(output)
	lines := strings.Split(output, "\n")
	cleanedLines := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			cleanedLines = append(cleanedLines, strings.TrimSpace(line))
		}
	}
	return strings.Join(cleanedLines, "\n")
}
