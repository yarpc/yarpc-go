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

package lib

import (
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	gogogen "github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"go.uber.org/yarpc/internal/protoplugin"
)

// _ActorUUIDFQN is the fully-qualified name of the FieldOptions extension
// that marks the actor UUID on a request field.
//
// The plugin recognises the option by name only. The option itself is
// defined in a separate (typically internal-monorepo) .proto file; this
// plugin does not take a build-time dependency on that definition. Whenever
// a target .proto transitively imports the file declaring this extension,
// the plugin discovers its field number from the descriptor request itself
// and emits ActorUUID() accessors for every annotated request type in the
// target file.
//
// Mirrors the auth.actor_uuid annotation handled by thriftrw-plugin-yarpc
// (see encoding/thrift/thriftrw-plugin-yarpc/uuid.go).
const _ActorUUIDFQN = "uber.auth.annotations.actor_uuid"

// fieldOptionsExtendee is the Extendee value that
// FieldDescriptorProto carries for any extension of google.protobuf.FieldOptions.
const fieldOptionsExtendee = ".google.protobuf.FieldOptions"

// uuidStepKind classifies one hop along a path from a request message to
// an actor_uuid-annotated leaf. The kind determines how the generated
// accessor traverses the hop: a nil-safe getter chain for scalar hops, or
// a range loop for container (repeated / map) hops.
type uuidStepKind int

const (
	// stepScalarMessage is a single (optional) message-typed hop. The
	// generated code chains a nil-safe GetXxx() and keeps walking on the
	// returned value.
	stepScalarMessage uuidStepKind = iota
	// stepRepeatedMessage is a `repeated Msg` hop. The generated code
	// ranges over the slice and recurses into every element.
	stepRepeatedMessage
	// stepMapMessage is a `map<K, Msg>` hop. The generated code ranges
	// over the map and recurses into every value.
	stepMapMessage
	// stepStringLeaf is a plain `string` leaf carrying the annotation. The
	// generated code appends the single value.
	stepStringLeaf
	// stepRepeatedStringLeaf is a `repeated string` leaf carrying the
	// annotation. The generated code appends every element of the slice.
	stepRepeatedStringLeaf
	// stepMapStringLeaf is a `map<K, string>` leaf carrying the
	// annotation. The generated code appends every value of the map. Map
	// iteration order in Go is non-deterministic, so the relative order of
	// values surfaced from a single map is unspecified.
	stepMapStringLeaf
)

// uuidPathStep is one hop along the path from a request message down to a
// leaf field carrying the annotation. Field names the field on the
// containing message; Kind classifies how the hop is traversed.
type uuidPathStep struct {
	Field *protoplugin.Field
	Kind  uuidStepKind
}

// uuidPath is a fully resolved chain from a request message down to one
// actor_uuid-annotated leaf. Steps is non-empty: the last entry is the
// leaf, earlier entries are the message-typed (possibly repeated/map)
// hops the chain walks through.
type uuidPath struct {
	Steps []uuidPathStep
}

// ActorUUID() body rendering modes. The plugin classifies each request
// into one of these and the template maps the mode to Go syntax, so the
// statement-level Go source (return / literal / var / append / range)
// lives in the template, not in hardcoded strings here.
const (
	// modeSlice: the request has a single path whose `repeated string`
	// leaf is reached through scalar hops only. The accessor returns that
	// field's slice directly: `return <SliceExpr>`.
	modeSlice = "slice"
	// modeLiteral: every path is scalar-only (a plain string leaf reached
	// through optional message hops). The accessor returns a composite
	// literal of the nil-safe getter chains: `return []string{Exprs...}`.
	modeLiteral = "literal"
	// modeBuilder: at least one path traverses a container (a repeated/map
	// message hop, or a map leaf), or scalars and slices mix across
	// paths. The accessor accumulates into `out` via Stmts and returns it.
	modeBuilder = "builder"
)

// actor_uuid builder statement kinds. Each maps to one line of Go the
// template renders; see the modeBuilder branch of the template.
const (
	// stmtAppend appends a single string expression: out = append(out, Expr).
	stmtAppend = "append"
	// stmtAppendSpread spreads a []string expression: out = append(out, Expr...).
	stmtAppendSpread = "appendSpread"
	// stmtRangeOpen opens a loop: for _, Var := range Expr {.
	stmtRangeOpen = "rangeOpen"
	// stmtClose closes the most recently opened loop: }.
	stmtClose = "close"
)

// actorUUIDStmt is one statement of a modeBuilder accessor body, kept as
// structured data so the template (not this package) owns the Go syntax.
type actorUUIDStmt struct {
	Kind string // one of stmtAppend / stmtAppendSpread / stmtRangeOpen / stmtClose
	Expr string // append value, spread slice, or range source; empty for stmtClose
	Var  string // loop variable; set only for stmtRangeOpen
}

// actorUUIDMethod describes a single ActorUUID() emission on a request
// message. The template iterates a slice of these to emit one accessor
// per request type that has at least one path to an annotated field.
//
// Mode selects how the template renders the body; only the fields
// relevant to that mode are populated:
//   - modeSlice   -> SliceExpr
//   - modeLiteral -> Exprs
//   - modeBuilder -> Stmts
type actorUUIDMethod struct {
	// GoTypeName is the Go-name of the request message in the package
	// being generated (e.g. "DeleteUserRequest"; "Foo_Bar" for nested).
	GoTypeName string
	// Mode is one of modeSlice / modeLiteral / modeBuilder.
	Mode string
	// SliceExpr is the getter chain returned directly in modeSlice.
	SliceExpr string
	// Exprs are the nil-safe getter chains listed in the modeLiteral
	// composite literal, in declaration order.
	Exprs []string
	// Stmts are the accumulator statements rendered in modeBuilder, in
	// order, with loops already balanced (open/close).
	Stmts []actorUUIDStmt
}

// actorUUIDMethods returns one ActorUUID() emission per request message of
// services in the target file that has at least one path to an
// actor_uuid-annotated string field. The same request type used by
// multiple methods is deduped (Go would otherwise refuse to compile two
// methods with the same name on the same receiver).
//
// Returns nil (and no error) when the target file's import graph does not
// reach the option's declaration, which is the common case and means the
// plugin should not emit any accessors.
func actorUUIDMethods(info *protoplugin.TemplateInfo) ([]*actorUUIDMethod, error) {
	num := findActorUUIDFieldNumber(info.File)
	if num == 0 {
		return nil, nil
	}
	ctx := newUUIDContext(info.File)
	pkgPath := info.File.GoPackage.Path
	seen := map[*protoplugin.Message]bool{}
	var out []*actorUUIDMethod
	for _, svc := range info.File.Services {
		for _, m := range svc.Methods {
			req := m.RequestType
			if req == nil {
				continue
			}
			// Methods can only be added to a Go type from the package
			// that declares it; cross-package request types must get
			// their accessor when their own file is generated.
			if req.File != info.File {
				continue
			}
			if seen[req] {
				continue
			}
			seen[req] = true

			paths := walkForUUID(req.Fields, num, ctx, map[*protoplugin.Message]bool{req: true})
			if len(paths) == 0 {
				continue
			}
			out = append(out, newActorUUIDMethod(req.GoType(pkgPath), paths))
		}
	}
	return out, nil
}

// newActorUUIDMethod classifies the given non-empty set of paths into a
// rendering mode and packages the data the template needs. It holds no Go
// statement syntax: the actual `return`/`[]string{}`/`append`/`for` text
// is emitted by the template from the Mode and the structured fields.
//
// The mode is chosen so the generated body is as simple as the shape
// allows:
//   - modeSlice: a single path whose `repeated string` leaf is reached
//     through scalar hops only. The getter chain already yields exactly
//     the []string wanted, so the accessor returns it directly. The
//     returned slice aliases the message's backing array, matching the
//     stock gogo getter it delegates to.
//   - modeLiteral: every path is scalar-only (a plain string leaf reached
//     through optional message hops). Each contributes one nil-safe getter
//     chain to a single composite literal. This is the common case.
//   - modeBuilder: any path traverses a container (repeated/map message
//     hop, or a map leaf), or scalars and slices mix across paths, so a
//     single expression cannot express the result (Go cannot spread a
//     slice or range a map inside `[]string{...}`). Falls back to an
//     accumulator with append/range statements.
func newActorUUIDMethod(goType string, paths []*uuidPath) *actorUUIDMethod {
	m := &actorUUIDMethod{GoTypeName: goType}
	if len(paths) == 1 {
		if expr, ok := slicePathExpr(paths[0]); ok {
			m.Mode = modeSlice
			m.SliceExpr = expr
			return m
		}
	}
	exprs := make([]string, 0, len(paths))
	allScalar := true
	for _, p := range paths {
		expr, ok := scalarPathExpr(p)
		if !ok {
			allScalar = false
			break
		}
		exprs = append(exprs, expr)
	}
	if allScalar {
		m.Mode = modeLiteral
		m.Exprs = exprs
		return m
	}
	m.Mode = modeBuilder
	for _, p := range paths {
		m.Stmts = append(m.Stmts, buildActorUUIDStmts(p)...)
	}
	return m
}

// getterChain returns the nil-safe gogo getter for f invoked on receiver
// expression recv, e.g. getterChain("t", actorField) -> "t.GetActor()".
// gogo's GetXxx() returns the zero value on a nil receiver, so chaining
// these stays panic-free through missing intermediate hops.
func getterChain(recv string, f *protoplugin.Field) string {
	return recv + ".Get" + gogogen.CamelCase(f.GetName()) + "()"
}

// scalarPathExpr returns the single nil-safe getter-chain expression for a
// scalar-only path - one whose every message hop is optional (scalar) and
// whose leaf is a plain string field. It reports false when the path
// traverses any container (repeated / map) hop or leaf, in which case the
// caller must emit loop-based statements instead.
func scalarPathExpr(p *uuidPath) (string, bool) {
	expr := "t"
	for _, s := range p.Steps {
		getter := getterChain(expr, s.Field)
		switch s.Kind {
		case stepScalarMessage:
			expr = getter
		case stepStringLeaf:
			return getter, true
		default:
			return "", false
		}
	}
	return "", false
}

// slicePathExpr returns the nil-safe getter-chain expression for a path
// whose every message hop is optional (scalar) and whose leaf is a
// `repeated string` field - i.e. the getter chain already yields exactly
// the []string the accessor wants (e.g. `t.GetActors()`). When this is the
// request's only path the accessor returns that slice directly, with no
// builder. It reports false for any path that traverses a repeated/map
// message hop or whose leaf is a map (those still need a loop).
func slicePathExpr(p *uuidPath) (string, bool) {
	expr := "t"
	for _, s := range p.Steps {
		getter := getterChain(expr, s.Field)
		switch s.Kind {
		case stepScalarMessage:
			expr = getter
		case stepRepeatedStringLeaf:
			return getter, true
		default:
			return "", false
		}
	}
	return "", false
}

// buildActorUUIDStmts lowers one path into the ordered, brace-balanced
// statements a modeBuilder accessor runs against its `out` accumulator to
// collect every actor_uuid value reachable along p from receiver "t".
//
// Consecutive scalar message hops collapse into a single nil-safe getter
// chain (a missing intermediate hop yields "" rather than panicking). A
// container hop (repeated / map) opens a `range` loop and the walk
// continues on the loop variable, so nested containers nest loops. Ranging
// over a nil slice or map is a no-op, keeping the whole body nil-safe.
//
// The result is structured data (no Go syntax): the template turns each
// actorUUIDStmt into a line of code.
func buildActorUUIDStmts(p *uuidPath) []actorUUIDStmt {
	var stmts []actorUUIDStmt
	expr := "t"
	varN := 0
	open := 0
	for _, s := range p.Steps {
		getter := getterChain(expr, s.Field)
		switch s.Kind {
		case stepScalarMessage:
			expr = getter
		case stepRepeatedMessage, stepMapMessage:
			varN++
			v := fmt.Sprintf("e%d", varN)
			stmts = append(stmts, actorUUIDStmt{Kind: stmtRangeOpen, Var: v, Expr: getter})
			expr = v
			open++
		case stepStringLeaf:
			stmts = append(stmts, actorUUIDStmt{Kind: stmtAppend, Expr: getter})
		case stepRepeatedStringLeaf:
			stmts = append(stmts, actorUUIDStmt{Kind: stmtAppendSpread, Expr: getter})
		case stepMapStringLeaf:
			varN++
			v := fmt.Sprintf("v%d", varN)
			stmts = append(stmts,
				actorUUIDStmt{Kind: stmtRangeOpen, Var: v, Expr: getter},
				actorUUIDStmt{Kind: stmtAppend, Expr: v},
				actorUUIDStmt{Kind: stmtClose},
			)
		}
	}
	for ; open > 0; open-- {
		stmts = append(stmts, actorUUIDStmt{Kind: stmtClose})
	}
	return stmts
}

// walkForUUID walks `fields` depth-first and returns every reachable
// actor_uuid-annotated leaf as its own path. The returned slice is empty
// (nil) when no path exists. It is the proto analogue of
// thriftrw-plugin-yarpc's findUUIDPath.
//
// A leaf is string-valued: a plain `string`, a `repeated string`, or a
// `map<K, string>` field carrying the annotation. The walker also
// descends through message-typed hops - scalar `Msg`, `repeated Msg`, and
// `map<K, Msg>` - collecting every annotated leaf underneath. Annotations
// on any other shape (non-string scalars, repeated non-string scalars, or
// a message-typed field itself) are silently ignored, though the walker
// still descends into message-typed fields looking for string leaves
// inside.
//
// All matches are collected, in declaration order: every annotated leaf
// reachable from `fields` contributes one path. Reaching more than one
// annotation is not an error.
//
// `visited` tracks message identities currently on the recursion stack;
// callers must seed it with the message owning `fields` so a self-cycle
// at the root level cannot loop forever. Because an entry is removed from
// `visited` once its subtree is fully walked, the same message type
// reached through two distinct sibling fields is walked for both (each
// route yields its own path), while a true cycle back to an ancestor
// still on the stack is pruned. Cycle detection spans container hops just
// as it does scalar ones.
func walkForUUID(
	fields []*protoplugin.Field,
	num int32,
	ctx *uuidContext,
	visited map[*protoplugin.Message]bool,
) []*uuidPath {
	var out []*uuidPath
	for _, f := range fields {
		out = append(out, walkUUIDField(f, num, ctx, visited)...)
	}
	return out
}

// walkUUIDField returns the actor_uuid paths contributed by a single field
// f (nil when it contributes none). It dispatches on the field's shape:
// a string-valued leaf yields one path; a message-typed field is descended
// into (recursing through walkForUUID), with map fields handled specially
// because proto encodes them as a synthetic entry message. See walkForUUID
// for the leaf/hop and cycle-detection semantics.
func walkUUIDField(
	f *protoplugin.Field,
	num int32,
	ctx *uuidContext,
	visited map[*protoplugin.Message]bool,
) []*uuidPath {
	isRepeated := f.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REPEATED

	if f.GetType() != descriptor.FieldDescriptorProto_TYPE_MESSAGE {
		// Scalar field. Only a string (or repeated string) carrying the
		// annotation is a valid leaf; any other scalar is ignored.
		if f.GetType() != descriptor.FieldDescriptorProto_TYPE_STRING || !hasActorUUID(f.GetOptions(), num) {
			return nil
		}
		kind := stepStringLeaf
		if isRepeated {
			kind = stepRepeatedStringLeaf
		}
		return []*uuidPath{leafPath(f, kind)}
	}

	inner := ctx.lookupMessage(f.GetTypeName())
	if inner == nil {
		return nil
	}

	// A map<K, V> is encoded as a repeated synthetic entry message
	// carrying map_entry = true with fields key (1) and value (2).
	if inner.GetOptions().GetMapEntry() {
		return walkUUIDMapField(f, inner, num, ctx, visited)
	}

	// Plain message hop, scalar or repeated.
	kind := stepScalarMessage
	if isRepeated {
		kind = stepRepeatedMessage
	}
	return descendUUID(f, inner, kind, num, ctx, visited)
}

// walkUUIDMapField returns the actor_uuid paths contributed by a map field
// f whose synthetic entry message is `entry`. A map<K, string> surfaces its
// values as a single leaf when annotated; a map<K, Msg> is descended into.
func walkUUIDMapField(
	f *protoplugin.Field,
	entry *protoplugin.Message,
	num int32,
	ctx *uuidContext,
	visited map[*protoplugin.Message]bool,
) []*uuidPath {
	val := mapValueField(entry)
	if val == nil {
		return nil
	}
	switch val.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		// map<K, string>: the values are the leaves.
		if !hasActorUUID(f.GetOptions(), num) {
			return nil
		}
		return []*uuidPath{leafPath(f, stepMapStringLeaf)}
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		// map<K, Msg>: recurse into the value message.
		return descendUUID(f, ctx.lookupMessage(val.GetTypeName()), stepMapMessage, num, ctx, visited)
	}
	return nil
}

// descendUUID walks `inner` (the message reached through f) for annotated
// leaves and prepends a hop of the given kind to every path found. It is
// the shared body of every message-typed hop (scalar, repeated, map). The
// cycle guard keeps `inner` on the recursion stack only while its subtree
// is walked, so sibling routes to the same type are each explored while a
// true back-edge to an ancestor is pruned. Returns nil when inner is nil
// or already on the stack.
func descendUUID(
	f *protoplugin.Field,
	inner *protoplugin.Message,
	kind uuidStepKind,
	num int32,
	ctx *uuidContext,
	visited map[*protoplugin.Message]bool,
) []*uuidPath {
	if inner == nil || visited[inner] {
		return nil
	}
	visited[inner] = true
	subs := walkForUUID(inner.Fields, num, ctx, visited)
	delete(visited, inner)
	return appendWithStep(nil, subs, uuidPathStep{Field: f, Kind: kind})
}

// leafPath builds a single-step path whose only hop is the annotated leaf
// field f.
func leafPath(f *protoplugin.Field, kind uuidStepKind) *uuidPath {
	return &uuidPath{Steps: []uuidPathStep{{Field: f, Kind: kind}}}
}

// appendWithStep prepends `step` to every sub-path of `subs` and appends
// the resulting paths to `out`, returning the grown slice. It is the
// shared splice used by every message-typed hop (scalar, repeated, map).
func appendWithStep(out []*uuidPath, subs []*uuidPath, step uuidPathStep) []*uuidPath {
	for _, sub := range subs {
		steps := make([]uuidPathStep, 0, 1+len(sub.Steps))
		steps = append(steps, step)
		steps = append(steps, sub.Steps...)
		out = append(out, &uuidPath{Steps: steps})
	}
	return out
}

// mapValueField returns the `value` field (field number 2) of a synthetic
// proto map entry message, or nil if it cannot be found. The walker uses
// the value field's type to decide whether a map is a string leaf or a
// message hop.
func mapValueField(entry *protoplugin.Message) *protoplugin.Field {
	for _, f := range entry.Fields {
		if f.GetNumber() == 2 || f.GetName() == "value" {
			return f
		}
	}
	return nil
}

// uuidContext is a per-generation resolver from a message's
// fully-qualified proto name (e.g. ".pkg.Outer.Inner") to its
// *protoplugin.Message. The plugin's registry already has this map but
// does not expose it, so the UUID walker rebuilds an equivalent index
// once at the start of every generation request and reuses it for every
// recursion step.
type uuidContext struct {
	msgByFQMN map[string]*protoplugin.Message
}

func newUUIDContext(file *protoplugin.File) *uuidContext {
	c := &uuidContext{msgByFQMN: map[string]*protoplugin.Message{}}
	c.indexFile(file)
	for _, dep := range file.TransitiveDependencies {
		c.indexFile(dep)
	}
	return c
}

// indexFile registers every (top-level and nested) message in f under its
// fully-qualified name. The protoplugin registry already flattens nested
// types into f.Messages, so a single pass is enough.
func (c *uuidContext) indexFile(f *protoplugin.File) {
	for _, m := range f.Messages {
		c.msgByFQMN[m.FQMN()] = m
	}
}

func (c *uuidContext) lookupMessage(typeName string) *protoplugin.Message {
	if typeName == "" {
		return nil
	}
	return c.msgByFQMN[typeName]
}

// findActorUUIDFieldNumber walks the target file and its transitive
// dependencies for an extension of google.protobuf.FieldOptions whose
// fully-qualified name is _ActorUUIDFQN, and returns its field number.
//
// Returns 0 if the option is not in scope (the .proto being generated did
// not transitively import the file declaring the option), in which case
// there is nothing to do.
func findActorUUIDFieldNumber(f *protoplugin.File) int32 {
	if n := findActorUUIDFieldNumberInFile(f); n != 0 {
		return n
	}
	for _, dep := range f.TransitiveDependencies {
		if n := findActorUUIDFieldNumberInFile(dep); n != 0 {
			return n
		}
	}
	return 0
}

func findActorUUIDFieldNumberInFile(f *protoplugin.File) int32 {
	pkg := f.GetPackage()
	if n := findActorUUIDInExtensions(pkg, f.GetExtension()); n != 0 {
		return n
	}
	for _, msg := range f.GetMessageType() {
		if n := findActorUUIDInNestedExtensions(pkg, msg); n != 0 {
			return n
		}
	}
	return 0
}

// findActorUUIDInExtensions scans a flat slice of extension declarations
// whose containing scope is `scope` (the proto package or a containing
// message FQN) for one that extends FieldOptions and matches _ActorUUIDFQN.
func findActorUUIDInExtensions(scope string, exts []*descriptor.FieldDescriptorProto) int32 {
	for _, ext := range exts {
		if ext.GetExtendee() != fieldOptionsExtendee {
			continue
		}
		fqn := ext.GetName()
		if scope != "" {
			fqn = scope + "." + fqn
		}
		if fqn == _ActorUUIDFQN {
			return ext.GetNumber()
		}
	}
	return 0
}

// findActorUUIDInNestedExtensions recurses into a message's nested
// extension declarations.
func findActorUUIDInNestedExtensions(parentScope string, msg *descriptor.DescriptorProto) int32 {
	scope := parentScope
	if name := msg.GetName(); name != "" {
		if scope == "" {
			scope = name
		} else {
			scope = scope + "." + name
		}
	}
	if n := findActorUUIDInExtensions(scope, msg.GetExtension()); n != 0 {
		return n
	}
	for _, nested := range msg.GetNestedType() {
		if n := findActorUUIDInNestedExtensions(scope, nested); n != 0 {
			return n
		}
	}
	return 0
}

// hasActorUUID reports whether opts carries the FieldOptions extension
// with the given field number set to true.
//
// The plugin must operate without a build-time dependency on the Go
// package generated for the extension. To do that:
//
//   - If gogo proto already has an ExtensionDesc registered at this field
//     number on FieldOptions (because some imported package happens to
//     have registered it - e.g. a test binary, or a downstream binary
//     that does import the option's .pb.go), reuse it. gogo rejects two
//     non-identical descriptors at the same field number ("proto:
//     descriptor conflict"), so we have no choice but to defer to the
//     registered one.
//   - Otherwise build an ExtensionDesc on the fly that matches what the
//     option's .pb.go would have generated (varint-typed bool at the
//     given field number) and cache it for the lifetime of the process.
//     Reusing the same *ExtensionDesc on every call is required: gogo
//     proto caches the descriptor pointer on first GetExtension and
//     refuses to accept a different pointer at the same field number on
//     later calls.
func hasActorUUID(opts *descriptor.FieldOptions, num int32) bool {
	if opts == nil || num == 0 {
		return false
	}
	desc := actorUUIDExtensionDesc(num)
	v, err := proto.GetExtension(opts, desc)
	if err != nil {
		return false
	}
	b, ok := v.(*bool)
	return ok && b != nil && *b
}

// actorUUIDExtensionCache memoises the *ExtensionDesc used to read the
// actor_uuid extension at a given field number. Only ever holds a single
// entry because per-process there is only ever one actor_uuid extension.
var (
	actorUUIDExtensionMu    sync.Mutex
	actorUUIDExtensionCache = map[int32]*proto.ExtensionDesc{}
)

func actorUUIDExtensionDesc(num int32) *proto.ExtensionDesc {
	actorUUIDExtensionMu.Lock()
	defer actorUUIDExtensionMu.Unlock()
	if desc, ok := actorUUIDExtensionCache[num]; ok {
		return desc
	}
	if desc := proto.RegisteredExtensions((*descriptor.FieldOptions)(nil))[num]; desc != nil {
		actorUUIDExtensionCache[num] = desc
		return desc
	}
	desc := &proto.ExtensionDesc{
		ExtendedType:  (*descriptor.FieldOptions)(nil),
		ExtensionType: (*bool)(nil),
		Field:         num,
		Name:          _ActorUUIDFQN,
		Tag:           fmt.Sprintf("varint,%d,opt,name=actor_uuid", num),
	}
	actorUUIDExtensionCache[num] = desc
	return desc
}
