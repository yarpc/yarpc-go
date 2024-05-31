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

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"
)

var (
	flagSet            = flag.NewFlagSet("service-test", flag.ExitOnError)
	flagDir            = flagSet.String("dir", "", "The relative directory to operate from, defaults to current directory")
	flagConfigFilePath = flagSet.String("file", "service-test.yaml", "The configuration file to use relative to the context directory")
	flagTimeout        = flagSet.Duration("timeout", 5*time.Second, "The time to wait until timing out")
	flagNoVerifyOutput = flagSet.Bool("no-validate-output", false, "Do not validate output and just run the commands")
	flagDebug          = flagSet.Bool("debug", false, "Log debug information")

	errSignal = errors.New("signal")
)

func main() {
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
	if err := do(
		*flagDir,
		*flagConfigFilePath,
		*flagTimeout,
		!(*flagNoVerifyOutput),
		*flagDebug,
	); err != nil {
		log.Fatal(err)
	}
}

func do(
	dir string,
	configFilePath string,
	timeout time.Duration,
	validateOutput bool,
	debug bool,
) (err error) {
	cmds, err := newCmds(configFilePath, dir, debug)
	if err != nil {
		return err
	}
	defer cleanupCmds(cmds, validateOutput, err)
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, os.Interrupt)
	go func() {
		for range signalC {
			cleanupCmds(cmds, validateOutput, errSignal)
		}
		os.Exit(1)
	}()

	errC := make(chan error)
	go func() { errC <- runCmds(cmds) }()
	select {
	case err := <-errC:
		if err != nil {
			return err
		}
	case <-time.After(timeout):
		return fmt.Errorf("timed out after %v", timeout)
	}
	if validateOutput {
		return validateCmds(cmds)
	}
	return nil
}

func newCmds(configFilePath string, dir string, debug bool) ([]*cmd, error) {
	config, err := newConfig(filepath.Join(dir, configFilePath))
	if err != nil {
		return nil, err
	}
	return config.Cmds(dir, debug)
}

func runCmds(cmds []*cmd) error {
	for i := 0; i < len(cmds)-1; i++ {
		cmd := cmds[i]
		if err := cmd.Start(); err != nil {
			return err
		}
		defer func() {
			cmd.Kill()
			_ = cmd.Wait()
		}()
	}
	cmd := cmds[len(cmds)-1]
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}

func validateCmds(cmds []*cmd) error {
	for _, cmd := range cmds {
		if err := cmd.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func cleanupCmds(cmds []*cmd, validateOutput bool, err error) {
	for _, cmd := range cmds {
		cmd.Clean(validateOutput && err == nil)
	}
}
