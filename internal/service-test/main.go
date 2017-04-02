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
	flagDir            = flag.String("dir", "", "The relative directory to operate from, defaults to current directory")
	flagConfigFilePath = flag.String("file", "service-test.yaml", "The configuration file to use relative to the context directory")
	flagTimeout        = flag.Duration("timeout", 5*time.Second, "The time to wait until timing out")
<<<<<<< HEAD
	flagNoVerifyOutput = flag.Bool("no-verify-output", false, "Do not verify output and just run the commands")
	flagVerbose        = flag.Bool("verbose", false, "Enable verbose logging")
=======
	flagNoVerifyOutput = flag.Bool("no-validate-output", false, "Do not validate output and just run the commands")
	flagDebug          = flag.Bool("debug", false, "Log debug information")
>>>>>>> dev

	errSignal = errors.New("signal")
)

func main() {
	flag.Parse()
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
<<<<<<< HEAD
	go func() {
		if serverCmd != nil {
			logCmd(serverCmd)
			if err := serverCmd.Start(); err != nil {
				errC <- fmt.Errorf("error starting server: %v", err)
				return
			}
			// kind of weird that we can timeout too
			// maybe add this to the timeout
			if config.SleepBeforeClientMs != 0 {
				<-time.After(time.Duration(config.SleepBeforeClientMs) * time.Millisecond)
			}
		}
		logCmd(clientCmd)
		if err := clientCmd.Start(); err != nil {
			errC <- fmt.Errorf("error starting client: %v", err)
			return
		}
		if err := clientCmd.Wait(); err != nil {
			errC <- fmt.Errorf("error on client wait: %v", err)
			return
		}
		if serverCmd != nil {
			killCmd(serverCmd)
			_ = serverCmd.Wait()
		}
		errC <- nil
	}()
=======
	go func() { errC <- runCmds(cmds) }()
>>>>>>> dev
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

<<<<<<< HEAD
func cleanupCmds(cmds ...*exec.Cmd) {
	for _, cmd := range cmds {
		killCmd(cmd)
	}
}

func killCmd(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		// https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

func logCmd(cmd *exec.Cmd) {
	verboseLogPrintf("%s %s", cmd.Path, strings.Join(cmd.Args, " "))
}

func validateRequiredEnvVars(requiredEnvVars []string) error {
	for _, requiredEnvVar := range requiredEnvVars {
		if os.Getenv(requiredEnvVar) == "" {
			return fmt.Errorf("environment variable %s must be set", requiredEnvVar)
=======
func runCmds(cmds []*cmd) error {
	for i := 0; i < len(cmds)-1; i++ {
		cmd := cmds[i]
		if err := cmd.Start(); err != nil {
			return err
>>>>>>> dev
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

func verboseLogPrintf(format string, args ...interface{}) {
	if *flagVerbose {
		log.Printf(format, args...)
	}
}
