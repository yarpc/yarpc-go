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

package yarpcconfig

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"go.uber.org/yarpc/internal/interpolate"
)

// Kit is an opaque object that carries context for the Configurator. Build
// functions that receive this object MUST NOT modify it.
type Kit struct {
	c *Configurator

	name string

	// Used to resolve interpolated variables.
	resolver interpolate.VariableResolver

	// TransportSpec currently being used. This may or may not be set.
	transportSpec *compiledTransportSpec
}

// Returns a shallow copy of this Kit with spec set to the given value.
func (k *Kit) withTransportSpec(spec *compiledTransportSpec) *Kit {
	newK := *k
	newK.transportSpec = spec
	return &newK
}

// ServiceName returns the name of the service for which components are being
// built.
func (k *Kit) ServiceName() string { return k.name }

var _typeOfKit = reflect.TypeOf((*Kit)(nil))

func (k *Kit) maybePeerChooserSpec(name string) *compiledPeerChooserSpec {
	return k.c.knownPeerChoosers[name]
}

func (k *Kit) peerListSpec(name string) (*compiledPeerListSpec, error) {
	if spec := k.c.knownPeerLists[name]; spec != nil {
		return spec, nil
	}

	msg := fmt.Sprintf("no recognized peer list or chooser %q", name)
	if available := k.peerChooserAndListSpecNames(); len(available) > 0 {
		msg = fmt.Sprintf("%s; need one of %s", msg, strings.Join(available, ", "))
	}

	return nil, errors.New(msg)
}

func (k *Kit) peerChooserPreset(name string) (*compiledPeerChooserPreset, error) {
	if k.transportSpec == nil {
		// Currently, transportspec is set only if we're inside build*Outbound.
		return nil, errors.New(
			"invalid Kit: make sure you passed in the same Kit your Build function received")
	}

	if spec := k.transportSpec.PeerChooserPresets[name]; spec != nil {
		return spec, nil
	}

	available := make([]string, 0, len(k.transportSpec.PeerChooserPresets))
	for name := range k.transportSpec.PeerChooserPresets {
		available = append(available, name)
	}

	msg := fmt.Sprintf("no recognized peer chooser preset %q", name)
	if len(available) > 0 {
		msg = fmt.Sprintf("%s; need one of %s", msg, strings.Join(available, ", "))
	}

	return nil, errors.New(msg)
}

func (k *Kit) peerChooserAndListSpecNames() (names []string) {
	for name := range k.c.knownPeerLists {
		names = append(names, name)
	}
	for name := range k.c.knownPeerChoosers {
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
