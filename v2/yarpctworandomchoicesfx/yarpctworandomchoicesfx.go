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

package yarpctworandomchoicesfx

import (
	"fmt"

	"go.uber.org/config"
	"go.uber.org/fx"
	yarpc "go.uber.org/yarpc/v2"
	"go.uber.org/yarpc/v2/yarpctworandomchoices"
)

const (
	_name             = "yarpctworandomchoicesfx"
	_configurationKey = "yarpc.choosers.two-random-choices"
)

// Module produces a yarpctworandomchoices peer list.
var Module = fx.Options(
	fx.Provide(NewConfig),
	fx.Provide(NewList),
)

// Config is the configuration for constructing a set of two random choices
// peer.Choosers.
type Config struct {
	Choosers map[string]TwoRandomConfig `yaml:",inline"`
}

// TwoRandomConfig is the configuration for constructing a specific two random
// choices peer.Chooser.
type TwoRandomConfig struct {
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
	return ConfigResult{Config: c}, nil
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

	Choosers []yarpc.Chooser `group:"yarpcfx"`
	Lists    []yarpc.List    `group:"yarpcfx"`
}

// NewList produces `yarpctworandomchoices.List`s as `yarpc.Chooser`s and
// `yarpc.List`s.
func NewList(p ListParams) (ListResult, error) {
	var (
		choosers []yarpc.Chooser
		lists    []yarpc.List
	)
	for name, c := range p.Config.Choosers {
		dialer, ok := p.Provider.Dialer(c.Dialer)
		if !ok {
			return ListResult{}, fmt.Errorf("failed to resolve dialer %q", c.Dialer)
		}

		var opts []yarpctworandomchoices.ListOption
		if c.Capacity > 0 {
			opts = append(opts, yarpctworandomchoices.Capacity(c.Capacity))
		}

		list := yarpctworandomchoices.New(name, dialer, opts...)

		choosers = append(choosers, list)
		lists = append(lists, list)
	}
	return ListResult{
		Choosers: choosers,
		Lists:    lists,
	}, nil
}
