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

package config

import "reflect"

// Kit carries internal dependencies for building peer choosers.
// The kit gets threaded through transport, outbound, and inbound builders
// so they can thread the kit through functions like BuildChooser on a
// ChooserConfig.
type Kit struct {
	c *Configurator

	name string
}

// ServiceName returns the name of the service for which components are being
// built.
func (k *Kit) ServiceName() string { return k.name }

var _typeOfKit = reflect.TypeOf((*Kit)(nil))

func (k *Kit) binder(name string) *compiledBinderSpec {
	return k.c.knownBinders[name]
}

func (k *Kit) chooser(name string) *compiledChooserSpec {
	return k.c.knownChoosers[name]
}
