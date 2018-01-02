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

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

var (
	errConfigNil       = errors.New("config nil")
	errConfigRunNotSet = errors.New("config run not set")
)

type config struct {
	RequiredEnvVars []string     `json:"required_env_vars,omitempty" yaml:"required_env_vars,omitempty"`
	Run             []*cmdConfig `json:"run,omitempty" yaml:"run,omitempty"`
}

type cmdConfig struct {
	Command string `json:"command,omitempty" yaml:"command,omitempty"`
	SleepMs int    `json:"sleep_ms,omitempty" yaml:"sleep_ms,omitempty"`
	Input   string `json:"input,omitempty" yaml:"input,omitempty"`
	Output  string `json:"output,omitempty" yaml:"output,omitempty"`
}

func newConfig(configFilePath string) (*config, error) {
	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}
	config := &config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}
	if err := config.validate(); err != nil {
		return nil, err
	}
	return config, nil
}

func (c *config) Cmds(dir string, debug bool) ([]*cmd, error) {
	cmds := make([]*cmd, 0, len(c.Run))
	for _, cmdConfig := range c.Run {
		cmd, err := newCmd(cmdConfig, dir, debug)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, nil
}

func (c *config) validate() error {
	if c == nil {
		return errConfigNil
	}
	if len(c.Run) == 0 {
		return errConfigRunNotSet
	}
	for _, requiredEnvVar := range c.RequiredEnvVars {
		if os.Getenv(requiredEnvVar) == "" {
			return fmt.Errorf("environment variable %s must be set", requiredEnvVar)
		}
	}
	return nil
}
