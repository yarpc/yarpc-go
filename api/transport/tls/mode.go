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

package tls

import (
	"encoding"
	"fmt"
)

const (
	// Disabled TLS mode allows plaintext connections only.
	Disabled Mode = iota

	// Permissive TLS mode allows both TLS and plaintext connections.
	Permissive

	// Enforced TLS mode allows accepts TLS connections only.
	Enforced
)

var (
	_ fmt.Stringer             = (*Mode)(nil)
	_ encoding.TextUnmarshaler = (*Mode)(nil)
)

// Mode represents the TLS mode of the transport.
type Mode uint16

// UnmarshalText implements encoding.TextUnmarshaler.
func (t *Mode) UnmarshalText(text []byte) error {
	switch s := string(text); s {
	case "disabled":
		*t = Disabled
	case "permissive":
		*t = Permissive
	case "enforced":
		*t = Enforced
	default:
		return fmt.Errorf("unknown tls mode string: %s", string(text))
	}

	return nil
}
