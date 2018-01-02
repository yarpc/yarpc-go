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

package pally

import (
	"errors"
	"sync"
)

// DefaultLabelValue is used in place of empty label values.
const DefaultLabelValue = "default"

var (
	// Match the Prometheus error message.
	errInconsistentCardinality = errors.New("inconsistent label cardinality")

	_digesterPool = sync.Pool{New: func() interface{} {
		return &digester{make([]byte, 0, 128)}
	}}
)

// A digester creates a null-delimited byte slice from a series of variable
// label values. It's an efficient way to create map keys from metric names and
// labels.
type digester struct {
	bs []byte
}

// For optimal performance, be sure to free each digester.
func newDigester() *digester {
	d := _digesterPool.Get().(*digester)
	d.bs = d.bs[:0]
	return d
}

func (d *digester) add(s string) {
	if len(d.bs) > 0 {
		// separate labels with a null byte
		d.bs = append(d.bs, '\x00')
	}
	d.bs = append(d.bs, s...)
}

func (d *digester) digest() []byte {
	return d.bs
}

func (d *digester) free() {
	_digesterPool.Put(d)
}

// Labels describe the dimensions of a metric.
type Labels map[string]string

// IsValidName checks whether the supplied string is a valid metric and label
// name in both Prometheus and Tally.
//
// Tally and Prometheus each allow runes that the other doesn't, so Pally can
// accept only the common subset. For simplicity, we'd also like the rules for
// metric names and label names to be the same even if that's more restrictive
// than absolutely necessary.
//
// Tally allows anything matching the regexp `^[0-9A-z_\-]+$`. Prometheus
// allows the regexp `^[A-z_:][0-9A-z_:]*$` for metric names, and
// `^[A-z_][0-9A-z_]*$` for label names.
//
// The common subset is `^[A-z_][0-9A-z_]+$`.
func IsValidName(s string) bool {
	if len(s) == 0 {
		return false
	}

	switch c := s[0]; {
	case 'A' <= c && c <= 'Z':
		break
	case 'a' <= c && c <= 'z':
		break
	case c == '_':
		break
	default:
		return false
	}

	// Don't incur the expense of ranging over runes, since no multibyte UTF-8
	// characters are legal.
	for i := 1; i < len(s); i++ {
		c := s[i]
		switch {
		case '0' <= c && c <= '9':
			continue
		case 'A' <= c && c <= 'Z':
			continue
		case 'a' <= c && c <= 'z':
			continue
		case c == '_':
			continue
		default:
			return false
		}
	}

	return true
}

// IsValidLabelValue checks whether the supplied string is a valid label value
// in both Prometheus and Tally.
//
// Tally allows label values that match the regexp `^[0-9A-z_\-.]+$`.
// Prometheus allows any valid UTF-8 string.
func IsValidLabelValue(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Don't incur the expense of ranging over runes, since no multibyte UTF-8
	// characters are legal.
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case '0' <= c && c <= '9':
			continue
		case 'A' <= c && c <= 'Z':
			continue
		case 'a' <= c && c <= 'z':
			continue
		case c == '_' || c == '-' || c == '.':
			continue
		default:
			return false
		}
	}
	return true
}

// ScrubLabelValue replaces any invalid runes in the input string with '-'. If
// the input is already a valid label value (in both Prometheus and Tally),
// it's returned unchanged.
func ScrubLabelValue(s string) string {
	if IsValidLabelValue(s) {
		return s
	}
	if len(s) == 0 {
		return DefaultLabelValue
	}

	d := newDigester()
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case '0' <= c && c <= '9':
			d.bs = append(d.bs, c)
		case 'A' <= c && c <= 'Z':
			d.bs = append(d.bs, c)
		case 'a' <= c && c <= 'z':
			d.bs = append(d.bs, c)
		case c == '_' || c == '-' || c == '.':
			d.bs = append(d.bs, c)
		default:
			d.bs = append(d.bs, '-')
		}
	}
	scrubbed := string(d.bs)
	d.free()
	return scrubbed
}

func areValidLabelValues(ss []string) bool {
	for _, s := range ss {
		if !IsValidLabelValue(s) {
			return false
		}
	}
	return true
}

func scrubLabelValues(ss []string) []string {
	if areValidLabelValues(ss) {
		return ss
	}

	scrubbed := make([]string, len(ss))
	for i, dirty := range ss {
		scrubbed[i] = ScrubLabelValue(dirty)
	}
	return scrubbed
}
