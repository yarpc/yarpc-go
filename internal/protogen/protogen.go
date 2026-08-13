// Copyright (c) 2026 Uber Technologies, Inc.
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

// Package protogen holds the descriptor-library-agnostic core of
// annotation-driven accessor generation for protoc plugins.
//
// The yarpc protobuf plugins come in two flavors that differ only in
// which descriptor library they are built on: protoc-gen-yarpc-go wraps
// github.com/gogo/protobuf, protoc-gen-yarpc-go-v2 wraps
// github.com/golang/protobuf. Those are structurally identical but
// nominally distinct type families, so code written against one cannot
// accept the other. This package factors out everything that does not
// touch descriptors, so each plugin only supplies a small converter:
//
//	descriptors --(per-plugin converter)--> Node graph
//	Node graph  --(Walk)-->                 []*Path
//	[]*Path     --(Lower)-->                Accessor (template data)
//
// The converter owns all per-library concerns: classifying field shapes
// from label/type enums, resolving type references and synthetic map
// entries, reading the annotation through the library's extension
// machinery, and computing Go identifier names with the library's own
// CamelCase (so getter names always agree with the structs that library
// generates). The Walk and Lower stages own the semantics that must stay
// identical across plugins: cycle pruning, path-count capping, rendering
// mode selection, and loop/brace construction.
package protogen

// StepKind classifies one hop along a path from a message to an
// annotated leaf. The kind determines how the generated accessor
// traverses the hop: a nil-safe getter chain for scalar hops, or a range
// loop for container (repeated / map) hops.
type StepKind int

const (
	// StepScalarMessage is a single (optional) message-typed hop. The
	// generated code chains a nil-safe GetXxx() and keeps walking on the
	// returned value.
	StepScalarMessage StepKind = iota
	// StepRepeatedMessage is a `repeated Msg` hop. The generated code
	// ranges over the slice and recurses into every element.
	StepRepeatedMessage
	// StepMapMessage is a `map<K, Msg>` hop. The generated code ranges
	// over the map and recurses into every value.
	StepMapMessage
	// StepStringLeaf is a plain `string` leaf carrying the annotation.
	// The generated code appends the single value.
	StepStringLeaf
	// StepRepeatedStringLeaf is a `repeated string` leaf carrying the
	// annotation. The generated code appends every element of the slice.
	StepRepeatedStringLeaf
	// StepMapStringLeaf is a `map<K, string>` leaf carrying the
	// annotation. The generated code appends every value of the map. Map
	// iteration order in Go is non-deterministic, so the relative order
	// of values surfaced from a single map is unspecified.
	StepMapStringLeaf
)

// Node is one message in the neutral graph a converter builds from
// descriptors. Fields holds only the walk-relevant fields, in
// declaration order: message-typed hops and annotated string-valued
// leaves. Everything else (non-string scalars, unannotated strings,
// unresolvable type references) is omitted by the converter.
//
// Converters must memoize Nodes per source message so that two fields
// referencing the same message share one *Node: the walker keys cycle
// detection on Node identity.
type Node struct {
	Fields []*Field
}

// Field is one walk-relevant field of a Node.
type Field struct {
	// Name is the proto field name (e.g. "caller_actor_id"), carried for
	// diagnostics and tests.
	Name string
	// GoName is the Go name of the field as the plugin's generator
	// library computes it (e.g. "CallerActorId"); the generated getter
	// is "Get" + GoName + "()". The converter computes it with the same
	// CamelCase the library uses to name generated struct fields, so the
	// emitted accessor always matches the generated code.
	GoName string
	// Kind classifies the field's shape; see StepKind.
	Kind StepKind
	// Child is the target message for message-shaped kinds (the element
	// message for repeated hops, the value message for map hops); nil
	// for leaf kinds.
	Child *Node
}

// Step is one hop of a resolved Path, a value copy of the Field it was
// derived from (minus Child).
type Step struct {
	Name   string
	GoName string
	Kind   StepKind
}

// Path is a fully resolved chain from a message down to one annotated
// leaf. Steps is non-empty: the last entry is the leaf, earlier entries
// are the message-typed (possibly repeated/map) hops the chain walks
// through.
type Path struct {
	Steps []Step
}
