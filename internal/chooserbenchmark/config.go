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

package chooserbenchmark

import (
	"fmt"
	"io"
	"time"

	"go.uber.org/yarpc/api/peer"
)

// PeerListConstructor is one of options provided by user, determine which
// load balancing policy to use
type PeerListConstructor func(t peer.Transport) peer.ChooserList

// Config is contains configurations set by user, is the only necessary object
// user have to create in order to run a bench mark
type Config struct {
	ClientGroups []ClientGroup
	ServerGroups []ServerGroup
	Duration     time.Duration
	Output       io.Writer
}

// ClientGroup is the configuration for clients
type ClientGroup struct {
	Name        string
	Count       int
	RPS         int
	Constructor PeerListConstructor
}

// ServerGroup is the configuration for servers
type ServerGroup struct {
	Name           string
	Count          int
	Latency        time.Duration
	LogNormalSigma float64
}

func (config *Config) checkClientGroup() error {
	names := map[string]struct{}{}
	for _, group := range config.ClientGroups {
		if group.Name == "" {
			return fmt.Errorf("client group name is nil")
		}
		if val, ok := names[group.Name]; ok {
			return fmt.Errorf("client group name duplicated, name: %q", val)
		}
		names[group.Name] = struct{}{}
		if group.RPS < 0 {
			return fmt.Errorf("RPS field must be greater than 0, RPS: %d", group.RPS)
		}
		if group.Count <= 0 {
			return fmt.Errorf("number of clients must be greater than 0, client group count: %d", group.Count)
		}
	}
	return nil
}

func (config *Config) checkServerGroup() error {
	names := map[string]struct{}{}
	for _, group := range config.ServerGroups {
		if len(group.Name) == 0 {
			return fmt.Errorf("server group name is nil")
		}
		if val, ok := names[group.Name]; ok {
			return fmt.Errorf("server group name duplicated, name: %q", val)
		}
		names[group.Name] = struct{}{}
		if group.Latency < 0 {
			return fmt.Errorf("latency must not be smaller 0, latency: %v", group.Latency)
		}
		if group.Latency >= config.Duration {
			return fmt.Errorf("latency must be smaller than test duration, latency: %v, duration: %v",
				group.Latency, config.Duration)
		}
		if group.Count <= 0 {
			return fmt.Errorf("number of servers must be greater than 0, server group count: %d", group.Count)
		}
	}
	return nil
}

// Validate is a self checker that make sure your configuration is legal
// will be run as first step in benchmark main work flow
func (config *Config) Validate() error {
	if config.Duration <= 0 {
		return fmt.Errorf(`test duration must be greater than 0, current: %v`, config.Duration)
	}
	if err := config.checkClientGroup(); err != nil {
		return err
	}
	if err := config.checkServerGroup(); err != nil {
		return err
	}
	return nil
}
