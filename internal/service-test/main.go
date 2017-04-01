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
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mattn/go-shellwords"
	"gopkg.in/yaml.v2"
)

var (
	flagContextDir     = flag.String("dir", "", "The relative directory to operate from, defaults to current directory")
	flagConfigFilePath = flag.String("file", "service-test.yaml", "The configuration file to use relative to the context directory")
	flagTimeout        = flag.Duration("timeout", 5*time.Second, "The time to wait until timing out")
	flagNoVerifyOutput = flag.Bool("no-verify-output", false, "Do not verify output and just run the commands")
	flagDebug          = flag.Bool("debug", false, "Log debug information")

	errConfigNil           = errors.New("config nil")
	errClientCommandNotSet = errors.New("config client_command not set")
)

type config struct {
	RequiredEnvVars     []string `json:"required_env_vars,omitempty" yaml:"required_env_vars,omitempty"`
	ClientCommand       string   `json:"client_command,omitempty" yaml:"client_command,omitempty"`
	ServerCommand       string   `json:"server_command,omitempty" yaml:"server_command,omitempty"`
	SleepBeforeClientMs int      `json:"sleep_before_client_ms,omitempty" yaml:"sleep_before_client_ms,omitempty"`
	Input               string   `json:"input,omitempty" yaml:"input,omitempty"`
	Output              string   `json:"output,omitempty" yaml:"output,omitempty"`
}

func main() {
	flag.Parse()
	if err := do(*flagContextDir, *flagConfigFilePath, *flagTimeout, !(*flagNoVerifyOutput)); err != nil {
		log.Fatal(err)
	}
}

func do(contextDir string, configFilePath string, timeout time.Duration, verifyOutput bool) (err error) {
	configFilePath = filepath.Join(contextDir, configFilePath)
	config, err := readConfig(configFilePath)
	if err != nil {
		return err
	}
	if err := validateRequiredEnvVars(config.RequiredEnvVars); err != nil {
		return err
	}

	stdout := newLockedBuffer()
	stderr := newLockedBuffer()
	defer func() {
		if !verifyOutput || err != nil {
			if data := stdout.Bytes(); len(data) > 0 {
				fmt.Print(string(data))
			}
		}
		if data := stderr.Bytes(); len(data) > 0 {
			fmt.Print(string(data))
		}
	}()
	clientCmd, err := getCmd(config.ClientCommand)
	if err != nil {
		return err
	}
	clientCmd.Dir = contextDir
	clientCmd.Stdout = stdout
	clientCmd.Stderr = stderr
	var serverCmd *exec.Cmd
	if config.ServerCommand != "" {
		serverCmd, err = getCmd(config.ServerCommand)
		if err != nil {
			return err
		}
		serverCmd.Dir = contextDir
		serverCmd.Stdout = stdout
		serverCmd.Stderr = stderr
	}
	defer cleanupCmds(clientCmd, serverCmd)
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, os.Interrupt)
	go func() {
		for range signalC {
			cleanupCmds(clientCmd, serverCmd)
		}
		os.Exit(1)
	}()

	inputBuffer := bytes.NewBuffer(nil)
	if config.Input != "" {
		if _, err := inputBuffer.Write([]byte(config.Input)); err != nil {
			return err
		}
	}
	clientCmd.Stdin = inputBuffer

	errC := make(chan error)
	go func() {
		if serverCmd != nil {
			debugPrintf("Starting %s", cmdString(serverCmd))
			if err := serverCmd.Start(); err != nil {
				errC <- fmt.Errorf("error starting server: %v", err)
				return
			}
			debugPrintf("Started %s", cmdString(serverCmd))
			defer func() {
				if serverCmd != nil {
					killCmd(serverCmd)
					_ = serverCmd.Wait()
					debugPrintf("Finished %s", cmdString(serverCmd))
				}
			}()
			// kind of weird that we can timeout too
			// maybe add this to the timeout
			if config.SleepBeforeClientMs != 0 {
				sleepDuration := time.Duration(config.SleepBeforeClientMs) * time.Millisecond
				debugPrintf("Sleeping %v before starting %s", sleepDuration, cmdString(clientCmd))
				<-time.After(sleepDuration)
			}
		}
		debugPrintf("Starting %s", cmdString(clientCmd))
		if err := clientCmd.Start(); err != nil {
			errC <- fmt.Errorf("error starting client: %v", err)
			return
		}
		debugPrintf("Started %s, now waiting", cmdString(clientCmd))
		if err := clientCmd.Wait(); err != nil {
			errC <- fmt.Errorf("error on client wait: %v", err)
			return
		}
		debugPrintf("Finished %s", cmdString(clientCmd))
		errC <- nil
	}()
	select {
	case err := <-errC:
		if err != nil {
			return err
		}
	case <-time.After(timeout):
		return fmt.Errorf("client timed out after %v", timeout)
	}
	if verifyOutput {
		output := cleanOutput(string(stdout.Bytes()))
		expectedOutput := cleanOutput(config.Output)
		if output != expectedOutput {
			return fmt.Errorf("expected\n%s\ngot\n%s", expectedOutput, output)
		}
	}
	return nil
}

func getCmd(argsString string) (*exec.Cmd, error) {
	parser := shellwords.NewParser()
	parser.ParseEnv = true
	args, err := parser.Parse(argsString)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("Command %s evaulated to empty", argsString)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd, nil
}

func cleanupCmds(cmds ...*exec.Cmd) {
	for _, cmd := range cmds {
		killCmd(cmd)
	}
}

func killCmd(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		debugPrintf("Killing %s", cmdString(cmd))
		// https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

func validateRequiredEnvVars(requiredEnvVars []string) error {
	for _, requiredEnvVar := range requiredEnvVars {
		if os.Getenv(requiredEnvVar) == "" {
			return fmt.Errorf("environment variable %s must be set", requiredEnvVar)
		}
	}
	return nil
}

func readConfig(configFilePath string) (*config, error) {
	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}
	config := &config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	return config, nil
}

func validateConfig(config *config) error {
	if config == nil {
		return errConfigNil
	}
	if config.ClientCommand == "" {
		return errClientCommandNotSet
	}
	return nil
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

type lockedBuffer struct {
	buffer bytes.Buffer
	lock   sync.RWMutex
}

func newLockedBuffer() *lockedBuffer {
	return &lockedBuffer{}
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.buffer.Write(p)
}

func (l *lockedBuffer) Bytes() []byte {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.buffer.Bytes()
}

func debugPrintf(format string, args ...interface{}) {
	if *flagDebug {
		log.Printf(format, args...)
	}
}

func cmdString(cmd *exec.Cmd) string {
	//if len(cmd.Args) == 0 {
	//return cmd.Path
	//}
	//return strings.Join(cmd.Args, " ")
	return fmt.Sprintf("%+v", cmd)
}
