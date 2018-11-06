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

package yarpcroundrobinfx

import (
	"fmt"

	"go.uber.org/config"
	"go.uber.org/fx"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpcroundrobin"
)

const (
	_name             = "yarpcroundrobinfx"
	_configurationKey = "yarpc.peers.round-robin"
)

// Module produces a yarpcroundrobin peer list.
var Module = fx.Options(
	fx.Provide(NewConfig),
	fx.Provide(NewList),
)

// Config is the configuration for constructing a set of round-robin peer.Choosers.
type Config struct {
	Clients map[string]RoundRobinConfig `yaml:",inline"`
}

// RoundRobinConfig is the configuration for constructing a specific round-robin peer.Chooser.
type RoundRobinConfig struct {
	Dialer   string `yaml:"dialer"`
	Capacity int    `yaml:"capacity"`
}

// ConfigParams defines the dependencies of this module.
type ConfigParams struct {
	fx.In

	Provider config.Provider
}

// ConfigResult defines the values produced by this module.
type ConfigResult struct {
	fx.Out

	Config Config
}

// NewConfig produces a Config.
func NewConfig(p ConfigParams) (ConfigResult, error) {
	c := Config{}
	if err := p.Provider.Get(_configurationKey).Populate(&c); err != nil {
		return ConfigResult{}, err
	}
	return ConfigResult{
		Config: c,
	}, nil
}

// ListParams defines the dependencies of this module.
type ListParams struct {
	fx.In

	Config   Config
	Provider yarpc.DialerProvider
}

// ListResult defines the values produced by this module.
type ListResult struct {
	fx.Out

	Choosers []yarpc.NamedChooser `group:"yarpcfx"`
	Lists    []yarpc.NamedList    `group:"yarpcfx"`
}

// NewList produces a a yarpcroundrobin.List into
// the yarpc.NamedList and yarpc.NamedChooser groups.
func NewList(p ListParams) (ListResult, error) {
	var (
		choosers []yarpc.NamedChooser
		lists    []yarpc.NamedList
	)
	for name, c := range p.Config.Clients {
		dialer, ok := p.Provider.Dialer(c.Dialer)
		if !ok {
			return ListResult{}, fmt.Errorf("failed to resolve dialer %q", c.Dialer)
		}

		var opts []yarpcroundrobin.ListOption
		if c.Capacity != 0 {
			opts = append(opts, yarpcroundrobin.Capacity(c.Capacity))
		}

		list := yarpcroundrobin.New(dialer, opts...)
		choosers = append(
			choosers,
			yarpc.NamedChooser{
				Name:    name,
				Chooser: yarpc.Chooser(list),
			},
		)
		lists = append(
			lists,
			yarpc.NamedList{
				Name: name,
				List: yarpc.List(list),
			},
		)
	}
	return ListResult{
		Choosers: choosers,
		Lists:    lists,
	}, nil
}
