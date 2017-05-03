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

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Kit carries internal dependencies for building peer lists.
// The kit gets threaded through transport, outbound, and inbound builders
// so they can thread the kit through functions like BuildPeerList on a
// PeerListConfig.
type Kit struct {
	c *Configurator

	name string
}

// ServiceName returns the name of the service for which components are being
// built.
func (k *Kit) ServiceName() string { return k.name }

var _typeOfKit = reflect.TypeOf((*Kit)(nil))

func (k *Kit) peerListSpec(name string) (*compiledPeerListSpec, error) {
	if spec := k.c.knownPeerLists[name]; spec != nil {
		return spec, nil
	}

	msg := fmt.Sprintf("no recognized peer list %q", name)
	if available := k.peerListSpecNames(); len(available) > 0 {
		msg = fmt.Sprintf("%s; need one of %s", msg, strings.Join(available, ", "))
	}

	return nil, errors.New(msg)
}

func (k *Kit) peerListSpecNames() (names []string) {
	for name := range k.c.knownPeerLists {
		names = append(names, name)
	}
	sort.Strings(names)
	return
}

func (k *Kit) peerListUpdaterSpec(name string) *compiledPeerListUpdaterSpec {
	return k.c.knownPeerListUpdaters[name]
}

func (k *Kit) peerListUpdaterSpecNames() (names []string) {
	for name := range k.c.knownPeerListUpdaters {
		names = append(names, name)
	}
	sort.Strings(names)
	return
}
